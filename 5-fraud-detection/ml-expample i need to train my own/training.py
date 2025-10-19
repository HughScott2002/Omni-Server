import os
import uuid
from datetime import datetime, timezone
from typing import Any, Dict, Tuple

import matplotlib.pyplot as plt
import mlflow
import numpy as np
import pandas as pd
from sklearn.ensemble import GradientBoostingClassifier
from sklearn.metrics import (
    confusion_matrix,
    f1_score,
    precision_score,
    recall_score,
    roc_auc_score,
)
from sklearn.model_selection import train_test_split
from sqlmodel import select
from sqlmodel.ext.asyncio.session import AsyncSession

from backend.app.core.logging import get_logger
from backend.app.core.ml.config import ml_settings
from backend.app.core.ml.feature_engineering import prepare_training_dataset
from backend.app.core.ml.models import MLModel, ModelStatusEnum, TrainingDataset

logger = get_logger()


class ModelTrainer:
    def __init__(self, session: AsyncSession):
        self.session = session

        mlflow.set_tracking_uri(ml_settings.MLFLOW_TRACKING_URI)

        try:
            experiment = mlflow.get_experiment_by_name(
                ml_settings.MLFLOW_EXPERIMENT_NAME
            )
            if experiment:
                self.experiment_id = experiment.experiment_id
            else:
                self.experiment_id = mlflow.create_experiment(
                    ml_settings.MLFLOW_EXPERIMENT_NAME
                )
            logger.info(
                f"Using MLflow experiment: {ml_settings.MLFLOW_EXPERIMENT_NAME}"
            )

        except Exception as e:
            logger.error(f"Failed to setup MLflow experiment: {e}")
            self.experiment_id = None

    def _get_feature_importance(self, model, feature_names):
        try:
            importances = model.feature_importances_

            indices = np.argsort(importances)[::-1]

            plt.figure(figsize=(10, 6))
            plt.title("Feature Importances")

            plt.bar(range(len(indices)), importances[indices], align="center")

            plt.xticks(
                range(len(indices)), [feature_names[i] for i in indices], rotation=90
            )

            plt.tight_layout()

            temp_file = os.path.join(
                ml_settings.MODEL_STORAGE_PATH, "feature_importance.png"
            )

            plt.savefig(temp_file)

            plt.close()

            return temp_file, {
                feature_names[i]: float(importances[i])
                for i in range(len(feature_names))
            }
        except Exception as e:
            logger.error(f"Error generating feature importance: {e}")
            return None, {}

    def _train_gradient_boosting(
        self,
        X_train: pd.DataFrame,
        y_train: pd.Series,
        hyperparams: Dict[str, Any] | None = None,
        run_id: str | None = None,
    ) -> Tuple[GradientBoostingClassifier, Dict[str, Any]]:
        if hyperparams is None:
            hyperparams = ml_settings.DEFAULT_GRADIENT_BOOSTING_PARAMS.copy()

        nested_run = None
        try:
            if run_id:
                nested_run = mlflow.start_run(run_id=run_id, nested=True)
            else:
                nested_run = mlflow.start_run(
                    experiment_id=self.experiment_id, nested=True
                )

            mlflow.log_params(hyperparams)

            model = GradientBoostingClassifier(**hyperparams)

            model.fit(X_train, y_train)

            mlflow.sklearn.log_model(
                model,
                "model",
                registered_model_name=ml_settings.MLFLOW_MODEL_REGISTRY_NAME,
            )

            importance_file, importance_dict = self._get_feature_importance(
                model, X_train.columns
            )

            for feature, importance in importance_dict.items():
                mlflow.log_metric(f"importance_{feature}", importance)

            if importance_file:
                mlflow.log_artifact(importance_file)

                try:
                    os.remove(importance_file)
                except Exception as e:
                    logger.warning(f"Failed to remove temporary file: {e}")

            model_uri = f"runs:/{nested_run.info.run_id}/model"

            return model, {"model_uri": model_uri, "run_id": nested_run.info.run_id}

        finally:
            if nested_run:
                mlflow.end_run()

    async def train_model(
        self,
        start_date: datetime,
        end_date: datetime,
        hyperparams: Dict[str, Any] | None = None,
    ) -> Tuple[MLModel, Dict[str, Any]]:

        if mlflow.active_run():
            mlflow.end_run()

        try:
            with mlflow.start_run(experiment_id=self.experiment_id) as run:
                run_id = run.info.run_id
                logger.info(f"Started MLflow run: {run_id}")

                mlflow.log_param("start_date", start_date.isoformat())
                mlflow.log_param("end_date", end_date.isoformat())
                mlflow.log_param("model_type", "gradient_boosting")

                logger.info("Preparing training dataset...")

                df = await prepare_training_dataset(
                    self.session, start_date, end_date, mlflow_run_id=run_id
                )

                if df.empty or len(df) < 10:
                    error_msg = f"Insufficient data for training: {len(df)} samples"

                    logger.error(error_msg)
                    mlflow.log_param("training_error", error_msg)
                    raise ValueError(error_msg)

                mlflow.log_metric("dataset_size", len(df))
                mlflow.log_metric("fraud_ratio", df["is_fraud"].mean())

                dataset_path = os.path.join(
                    ml_settings.DATASET_STORAGE_PATH, f"dataset_{run_id}.csv"
                )

                df.to_csv(dataset_path, index=False)

                mlflow.log_artifact(dataset_path, "dataset")

                dataset_record = TrainingDataset(
                    id=uuid.uuid4(),
                    name=f"fraud_dataset_{datetime.now(timezone.utc).strftime('%Y%m%d_%H%M%S')}",
                    version=f"{datetime.now(timezone.utc).strftime('%Y%m%d_%H%M%S')}",
                    start_date=start_date,
                    end_date=end_date,
                    total_samples=len(df),
                    fraud_samples=int(df["is_fraud"].sum()),
                    legitimate_samples=int(len(df) - df["is_fraud"].sum()),
                    dataset_path=dataset_path,
                    mlflow_artifact_uri=f"runs:/{run_id}/artifacts/dataset",
                    feature_info={
                        "columns": df.columns.to_list(),
                        "dtypes": {col: str(df[col].dtype) for col in df.columns},
                        "fraud_ratio": float(df["is_fraud"].mean()),
                    },
                )

                self.session.add(dataset_record)

                await self.session.commit()

                await self.session.refresh(dataset_record)

                X = df.drop(["is_fraud", "transaction_id"], axis=1, errors="ignore")

                y = df["is_fraud"]

                X_train, X_test, y_train, y_test = train_test_split(
                    X, y, test_size=0.2, random_state=42, stratify=y
                )

                mlflow.log_metric("train_size", len(X_train))
                mlflow.log_metric("test_size", len(X_test))

                logger.info("Training gradient boosting model...")
                model, run_info = self._train_gradient_boosting(
                    X_train, y_train, hyperparams, run_id
                )

                logger.info("Evaluating model performance...")

                y_pred = model.predict(X_test)

                y_prob = model.predict_proba(X_test)[:, 1]

                auc = roc_auc_score(y_test, y_prob)
                precision = precision_score(y_test, y_pred)
                recall = recall_score(y_test, y_pred)
                f1 = f1_score(y_test, y_pred)
                cm = confusion_matrix(y_test, y_pred)

                metrics = {
                    "auc": float(auc),
                    "precision": float(precision),
                    "recall": float(recall),
                    "f1": float(f1),
                    "true_negatives": int(cm[0, 0]),
                    "false_positives": int(cm[0, 1]),
                    "false_negatives": int(cm[1, 0]),
                    "true_postives": int(cm[1, 1]),
                }

                for name, value in metrics.items():
                    mlflow.log_metric(name, value)

                try:
                    version_num = 1

                    stmt = select(MLModel).order_by("created_at desc").limit(1)

                    result = await self.session.exec(stmt)

                    latest_model = result.first()

                    if latest_model:
                        try:
                            if latest_model.version.isdigit():
                                version_num = int(latest_model.version) + 1
                        except (ValueError, AttributeError):
                            version_num = 1
                except Exception as e:
                    logger.warning(f"Error determining model version: {e}")
                    version_num = 1

                version_str = str(version_num)

                model_record = MLModel(
                    id=uuid.uuid4(),
                    name=f"gradient_boosting_{datetime.now(timezone.utc).strftime('%Y%m%d_%H%M%S')}",
                    version=version_str,
                    status=ModelStatusEnum.READY,
                    auc_score=metrics["auc"],
                    precision=metrics["precision"],
                    recall=metrics["recall"],
                    f1_score=metrics["f1"],
                    features=X.columns.to_list(),
                    hyperparameters=hyperparams
                    or ml_settings.DEFAULT_GRADIENT_BOOSTING_PARAMS,
                    training_dataset_size=len(X_train),
                    mlflow_run_id=run_id,
                    mlflow_experiment_id=self.experiment_id,
                    mlflow_model_uri=run_info["model_uri"],
                    trained_at=datetime.now(timezone.utc),
                )

                self.session.add(model_record)
                await self.session.commit()
                await self.session.refresh(model_record)

                logger.info(
                    f"Successfully trained model {model_record.id} with AUC: {metrics["auc"]:.4f}"
                )

                mlflow.end_run()

                return model_record, metrics
        except Exception as e:
            logger.error(f"Error in trained_model: {e}")
            if mlflow.active_run():
                mlflow.end_run()
            raise
