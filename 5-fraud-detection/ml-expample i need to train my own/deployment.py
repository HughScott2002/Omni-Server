from datetime import datetime, timezone
from typing import Any, Dict, Optional, Tuple
from uuid import UUID

import mlflow
from sqlmodel import select
from sqlmodel.ext.asyncio.session import AsyncSession

from backend.app.core.ai.enums import AIReviewStatusEnum
from backend.app.core.logging import get_logger
from backend.app.core.ml.config import ml_settings
from backend.app.core.ml.models import MLModel, ModelPrediction, ModelStatusEnum
from backend.app.transaction.enums import TransactionStatusEnum
from backend.app.transaction.models import Transaction

logger = get_logger()


class ModelDeployer:

    def __init__(self, session: AsyncSession):
        self.session = session
        mlflow.set_tracking_uri(ml_settings.MLFLOW_TRACKING_URI)

    async def deploy_model(self, model_id: UUID) -> MLModel:
        model = await self.session.get(MLModel, model_id)
        if not model:
            raise ValueError(f"Model {model_id} not found")

        if model.status != ModelStatusEnum.READY:
            raise ValueError(
                f"Model {model_id} is not ready for deployment (status: {model.status})"
            )

        try:
            stmt = select(MLModel).where(MLModel.status == ModelStatusEnum.DEPLOYED)

            result = await self.session.exec(stmt)
            current_deployed = result.first()

            if current_deployed:
                current_deployed.status = ModelStatusEnum.ARCHIVED
                self.session.add(current_deployed)

                if current_deployed.mlflow_run_id:
                    try:
                        with mlflow.start_run(run_id=current_deployed.mlflow_run_id):
                            mlflow.log_param("deployment_status", "ARCHIVED")
                            mlflow.log_param(
                                "archived_at", datetime.now(timezone.utc).isoformat()
                            )

                    except Exception as e:
                        logger.warning(
                            f"Failed to update MLflow status for archived model: {e}"
                        )

                model.status = ModelStatusEnum.DEPLOYED
                model.deployed_at = datetime.now(timezone.utc)
                self.session.add(model)

                if model.mlflow_run_id:
                    try:
                        with mlflow.start_run(run_id=model.mlflow_run_id):
                            mlflow.log_param("deployment_status", "DEPLOYED")
                            mlflow.log_param(
                                "deployed_at", model.deployed_at.isoformat()
                            )

                            if model.mlflow_model_uri:
                                model_name = "fraud_detection_gradient_boosting"
                                client = mlflow.MlflowClient()
                                versions = client.get_latest_versions(model_name)

                                if versions:
                                    latest_version = versions[0].version
                                    client.transition_model_version_stage(
                                        name=model_name,
                                        version=latest_version,
                                        stage="Production",
                                    )
                                    logger.info(
                                        f"Transitioned model {model_name} version {latest_version} to Production"
                                    )
                    except Exception as e:
                        logger.warning(
                            f"Failed to update MLflow status for deployed model: {e}"
                        )
                await self.session.commit()

                await self.session.refresh(model)

                logger.info(
                    f"Deployed model {model_name} (version {model.version}) to production"
                )

            return model
        except Exception as e:
            logger.error(f"Error deploying model: {e}")
            await self.session.rollback()
            raise

    async def get_deployed_model(self) -> Optional[MLModel]:
        stmt = select(MLModel).where(MLModel.status == ModelStatusEnum.DEPLOYED)

        result = await self.session.exec(stmt)
        return result.first()


class ModelInference:
    def __init__(self, session: AsyncSession):
        self.session = session
        self.model_cache = {}
        mlflow.set_tracking_uri(ml_settings.MLFLOW_TRACKING_URI)

    async def predict_fraud(
        self, transaction: Transaction
    ) -> Tuple[float, Dict[str, Any]]:
        try:
            deployer = ModelDeployer(self.session)

            model_record = await deployer.get_deployed_model()

            if not model_record:
                logger.warning("No deployed model found, using fallback prediction")
                return await self._fallback_prediction(transaction)

            if not model_record.mlflow_model_uri:
                logger.warning(
                    f"Model {model_record.id} has no MLflow URI, using fallback"
                )
                return await self._fallback_prediction(transaction)

            from backend.app.core.ml.feature_engineering import FeatureExtractor

            feature_extractor = FeatureExtractor(self.session)

            features = await feature_extractor.extract_features_for_transaction(
                transaction
            )

            try:
                if model_record.id not in self.model_cache:
                    model = mlflow.sklearn.load_model(model_record.mlflow_model_uri)
                    self.model_cache[model_record.id] = model

                else:
                    model = self.model_cache[model_record.id]

                if "transaction_id" in features:
                    del features[features["transaction_id"]]

                import pandas as pd

                feature_df = pd.DataFrame([features])

                missing_cols = set(model.feature_names_in_) - set(feature_df.columns)
                for col in missing_cols:
                    feature_df[col] = 0

                feature_df = feature_df[model.feature_names_in_]

                fraud_probability = float(model.predict_proba(feature_df)[0, 1])

                feature_importance = {}

                if hasattr(model, "feature_importances_"):
                    importance_values = model.feature_importances_

                    for i, feature_name in enumerate(model.feature_names_in_):
                        if i < len(importance_values):
                            importance = float(importance_values[i])
                            feature_value = float(feature_df[feature_name].iloc[0])
                            contribution = importance * feature_value
                            if contribution > 0:
                                feature_importance[feature_name] = contribution
                top_features = dict(
                    sorted(
                        feature_importance.items(), key=lambda x: x[1], reverse=True
                    )[:10]
                )
            except Exception as e:
                logger.error(f"Error loading or using MLflow model: {e}")

                return await self._fallback_prediction(transaction)

            if model_record.mlflow_run_id:
                try:
                    with mlflow.start_run(run_id=model_record.mlflow_run_id):
                        mlflow.log_metric("prediction_count", 1, step=1)
                        mlflow.log_metric("prediction_score", fraud_probability)

                        mlflow.log_param("transaction_id", str(transaction.id))

                except Exception as e:
                    logger.warning(f"Failed to log prediction to MLflow: {e}")

            prediction = ModelPrediction(
                transaction_id=transaction.id,
                model_id=model_record.id,
                prediction_score=fraud_probability,
                input_features=features,
                mlflow_run_id=model_record.mlflow_run_id,
            )

            self.session.add(prediction)
            await self.session.commit()
            await self.session.refresh(prediction)

            prediction_details = {
                "model_name": model_record.name,
                "model_version": model_record.version,
                "model_id": str(model_record.id),
                "prediction_time": datetime.now(timezone.utc).isoformat(),
                "mlflow_run_id": model_record.mlflow_run_id,
                "risk_factors": top_features,
            }

            return fraud_probability, prediction_details
        except Exception as e:
            logger.error(f"Error during fraud prediction: {e}")
            return await self._fallback_prediction(transaction)

    async def _fallback_prediction(
        self, transaction: Transaction
    ) -> Tuple[float, Dict[str, Any]]:

        amount = float(transaction.amount)

        if amount > 10000:
            fraud_probability = 0.7
        elif amount > 5000:
            fraud_probability = 0.5
        elif amount > 1000:
            fraud_probability = 0.3
        else:
            fraud_probability = 0.1

        risk_factors = {"amount": amount}

        hour = transaction.created_at.hour

        is_business_hours = 9 <= hour <= 17

        risk_factors["outside_business_hours"] = 0 if is_business_hours else 0.2

        if hour < 6 or hour > 22:
            fraud_probability += 0.1
            risk_factors["late_night_transaction"] = 0.1

        fraud_probability = min(0.9, fraud_probability)

        return fraud_probability, {
            "model_name": "fallback_heuristic",
            "model_version": "v1",
            "prediction_time": datetime.now(timezone.utc).isoformat(),
            "is_fallback": True,
            "risk_factors": risk_factors,
        }


async def update_transaction_risk(
    transaction: Transaction,
    fraud_probability: float,
    risk_threshold: float,
    prediction_details: Dict[str, Any],
    session: AsyncSession,
) -> Transaction:

    if transaction.transaction_metadata is None:
        transaction.transaction_metadata = {}

    transaction.transaction_metadata["risk_assessment"] = {
        "score": fraud_probability,
        "threshold": risk_threshold,
        "is_high_risk": fraud_probability >= risk_threshold,
        "assessed_at": datetime.now(timezone.utc).isoformat(),
        "model_details": {
            "name": prediction_details.get("model_name", "unknown"),
            "version": prediction_details.get("model_version", "unknown"),
            "id": prediction_details.get("model_id", "unknown"),
            "mlflow_run_id": prediction_details.get("mlflow_run_id", None),
        },
    }

    if fraud_probability >= risk_threshold:
        transaction.ai_review_status = AIReviewStatusEnum.FLAGGED

        if transaction.status == TransactionStatusEnum.Pending:
            transaction.status = TransactionStatusEnum.Pending

    else:
        transaction.ai_review_status = AIReviewStatusEnum.CLEARED

    session.add(transaction)
    await session.commit()
    await session.refresh(transaction)

    return transaction
