from datetime import datetime, timedelta, timezone
from typing import Any, Dict, List, Optional
from uuid import UUID

import mlflow
from sqlmodel import desc, select
from sqlmodel.ext.asyncio.session import AsyncSession

from backend.app.core.ai.enums import AIReviewStatusEnum
from backend.app.core.logging import get_logger
from backend.app.core.ml.config import ml_settings
from backend.app.core.ml.models import MLModel, ModelPrediction
from backend.app.transaction.models import Transaction

logger = get_logger()

mlflow.set_tracking_uri(ml_settings.MLFLOW_TRACKING_URI)


class ModelEvaluator:
    def __init__(self, session: AsyncSession):
        self.session = session

        try:
            experiment = mlflow.get_experiment_by_name("model_comparisons")
            if experiment:
                self.experiment_id = experiment.experiment_id
            else:
                self.experiment_id = mlflow.create_experiment("model_comparisons")
        except Exception as e:
            logger.error(f"Failed to setup MLflow experiment: {e}")
            self.experiment_id = None

    async def _calculate_auc(self, predictions, actuals):
        try:
            from sklearn.metrics import roc_auc_score

            if len(set(actuals)) < 2:
                logger.warning("cannot calculate the AUC with less than 2 classes")
                return 0.5
            return float(roc_auc_score(actuals, predictions))
        except ImportError:
            logger.warning(
                "scikit-learn not available, using approximate AUC calculation"
            )

            n_pos = sum(actuals)
            n_neg = len(actuals) - n_pos

            if n_pos == 0 or n_neg == 0:
                return 0.5

            paired = sorted(zip(predictions, actuals), key=lambda x: x[0], reverse=True)

            ranks = [i + 1 for i in range(len(paired)) if paired[i][1] == 1]

            return (sum(ranks) - n_pos * (n_pos + 1) / 2) / (n_pos * n_neg)

    async def _calculate_recall(self, predictions, actuals):
        true_positives = sum(
            1 for p, a in zip(predictions, actuals) if p >= 0.5 and a == 1
        )
        actual_positives = sum(actuals)

        if actual_positives == 0:
            return 0.0

        return true_positives / actual_positives

    async def _calculate_f1(self, predictions, actuals):
        precision = await self._calculate_precision(predictions, actuals)
        recall = await self._calculate_recall(predictions, actuals)

        if precision + recall == 0:
            return 0.0

        return 2 * precision * recall / (precision + recall)

    async def _calculate_precision(self, predictions, actuals):
        predicted_positive = sum(1 for p in predictions if p >= 0.5)

        if predicted_positive == 0:
            return 0.0

        true_positives = sum(
            1 for p, a in zip(predictions, actuals) if p >= 0.5 and a == 1
        )

        return true_positives / predicted_positive

    async def _generate_confusion_matrix(self, predictions, actuals):
        return {
            "true_positives": sum(
                1 for p, a in zip(predictions, actuals) if p >= 0.5 and a == 1
            ),
            "false_positives": sum(
                1 for p, a in zip(predictions, actuals) if p >= 0.5 and a == 0
            ),
            "true_negatives": sum(
                1 for p, a in zip(predictions, actuals) if p < 0.5 and a == 0
            ),
            "false_negatives": sum(
                1 for p, a in zip(predictions, actuals) if p < 0.5 and a == 1
            ),
        }

    async def evaluate_model_performance(
        self,
        model_id: UUID,
        start_date: Optional[datetime] = None,
        end_date: Optional[datetime] = None,
    ) -> Dict[str, Any]:
        try:
            model = await self.session.get(MLModel, model_id)

            if not model:
                raise ValueError(f"Model {model_id} not found")

            if not end_date:
                end_date = datetime.now(timezone.utc)
            if not start_date:
                start_date = end_date - timedelta(days=30)

            if model.mlflow_run_id:
                with mlflow.start_run(run_id=model.mlflow_run_id):
                    mlflow.log_param("evaluation_start_date", start_date.isoformat())
                    mlflow.log_param("evaluation_end_date", end_date.isoformat())

            query = select(ModelPrediction).where(
                ModelPrediction.model_id == model_id,
                ModelPrediction.prediction_timestamp >= start_date,
                ModelPrediction.prediction_timestamp <= end_date,
            )

            result = await self.session.exec(query)
            predictions = result.all()

            transaction_ids = [p.transaction_id for p in predictions]

            if not transaction_ids:
                logger.warning(
                    f"No predictions found for model {model_id} in the specified time period"
                )
                return {
                    "model_id": str(model_id),
                    "model_name": model.name,
                    "model_version": model.version,
                    "evaluation_period": {
                        "start": start_date.isoformat(),
                        "end": end_date.isoformat(),
                    },
                    "metrics": {
                        "total_predictions": 0,
                        "error": "No predictions found in the specified time period",
                    },
                }

            transactions = []

            for tx_id in transaction_ids:
                tx_query = select(Transaction).where(Transaction.id == tx_id)

                tx_result = await self.session.exec(tx_query)
                tx = tx_result.first()
                if tx:
                    transactions.append(tx)

            transaction_status_map = {
                t.id: (t.ai_review_status == AIReviewStatusEnum.FLAGGED)
                for t in transactions
            }

            prediction_scores = [
                p.prediction_score
                for p in predictions
                if p.transaction_id in transaction_status_map
            ]

            actual_labels = [
                1 if transaction_status_map.get(p.transaction_id, False) else 0
                for p in predictions
                if p.transaction_id in transaction_status_map
            ]

            metrics = {
                "auc": await self._calculate_auc(prediction_scores, actual_labels),
                "precision": await self._calculate_precision(
                    prediction_scores, actual_labels
                ),
                "recall": await self._calculate_recall(
                    prediction_scores, actual_labels
                ),
                "f1": await self._calculate_f1(prediction_scores, actual_labels),
                "total_preidictions": len(predictions),
                "confusion_matrix": await self._generate_confusion_matrix(
                    prediction_scores, actual_labels
                ),
            }

            if model.mlflow_run_id:
                with mlflow.start_run(run_id=model.mlflow_run_id):
                    for metric_name, metric_value in metrics.items():
                        if isinstance(metric_value, (int, float)):
                            mlflow.log_metric(f"eval_{metric_name}", metric_value)
                    if "confusion_matrix" in metrics:
                        import json

                        cm_path = "/tmp/confusion_matrix.json"

                        with open(cm_path, "w") as f:
                            json.dump(metrics["confusion_matrix"], f)

                        mlflow.log_artifact(cm_path, "evaluation")

            return {
                "model_id": str(model_id),
                "model_name": model.name,
                "model_version": model.version,
                "evaluation_period": {
                    "start": start_date.isoformat(),
                    "end": end_date.isoformat(),
                },
                "metrics": metrics,
            }
        except Exception as e:
            logger.error(f"Error evaluating model performance: {e}")
            raise

    async def get_model_metrics_trend(
        self, model_id: UUID, days: int = 30
    ) -> List[Dict[str, Any]]:

        end_date = datetime.now(timezone.utc)

        start_date = end_date - timedelta(days=days)

        metrics_trend = []

        current_date = start_date

        while current_date <= end_date:
            day_start = current_date.replace(hour=0, minute=0, second=0, microsecond=0)
            day_end = day_start + timedelta(days=1) - timedelta(microseconds=1)

            try:
                daily_metrics = await self.evaluate_model_performance(
                    model_id=model_id, start_date=day_start, end_date=day_end
                )

                if daily_metrics["metrics"]["total_predictions"] > 0:
                    metrics_trend.append(
                        {
                            "date": day_start.strftime("%Y-%m-%d"),
                            "metrics": daily_metrics["metrics"],
                        }
                    )
            except Exception as e:
                logger.error(f"Error getting metrics for {day_start.date()}: {e}")

            current_date += timedelta(days=1)
        return metrics_trend

    async def get_false_positives(
        self,
        model_id: UUID,
        start_date: datetime | None = None,
        end_date: datetime | None = None,
        limit: int = 100,
    ) -> List[Dict[str, Any]]:

        if not end_date:
            end_date = datetime.now(timezone.utc)
        if not start_date:
            start_date = end_date - timedelta(days=30)

        stmt = (
            select(ModelPrediction, Transaction)
            .join(Transaction)
            .where(
                ModelPrediction.transaction_id == Transaction.id,
                ModelPrediction.model_id == model_id,
                ModelPrediction.prediction_timestamp >= start_date,
                ModelPrediction.prediction_timestamp <= end_date,
                ModelPrediction.prediction_score >= 0.5,
                Transaction.ai_review_status == AIReviewStatusEnum.CLEARED,
            )
            .order_by(desc(ModelPrediction.prediction_score))
            .limit(limit)
        )

        result = await self.session.exec(stmt)
        false_positives = result.all()

        formatted_results = []

        for prediction, tx in false_positives:
            formatted_results.append(
                {
                    "transaction_id": str(tx.id),
                    "reference": tx.reference,
                    "amount": str(tx.amount),
                    "prediction_score": prediction.prediction_score,
                    "transaction_date": tx.created_at.isoformat(),
                    "prediction_date": prediction.prediction_timestamp.isoformat(),
                    "features": prediction.input_features,
                    "metadata": tx.transaction_metadata,
                }
            )
        model = await self.session.get(MLModel, model_id)

        if model and model.mlflow_run_id and formatted_results:
            with mlflow.start_run(run_id=model.mlflow_run_id):
                mlflow.log_metric("false_positive_count", len(formatted_results))

                for i, fp in enumerate(formatted_results[:5]):
                    mlflow.log_metric(f"top_fp_{i+1}_score", fp["prediction_score"])

                import json

                fp_path = "/tmp/false_positives.json"

                with open(fp_path, "w") as f:
                    json.dump(formatted_results, f)

                mlflow.log_artifact(fp_path, "evaluation/false_positives")
        return formatted_results

    async def get_false_negatives(
        self,
        model_id: UUID,
        start_date: datetime | None = None,
        end_date: datetime | None = None,
        limit: int = 100,
    ) -> List[Dict[str, Any]]:

        if not end_date:
            end_date = datetime.now(timezone.utc)
        if not start_date:
            start_date = end_date - timedelta(days=30)

        stmt = (
            select(ModelPrediction, Transaction)
            .join(Transaction)
            .where(
                ModelPrediction.transaction_id == Transaction.id,
                ModelPrediction.model_id == model_id,
                ModelPrediction.prediction_timestamp >= start_date,
                ModelPrediction.prediction_timestamp <= end_date,
                ModelPrediction.prediction_score < 0.5,
                Transaction.ai_review_status == AIReviewStatusEnum.CONFIRMED_FRAUD,
            )
            .order_by(desc(ModelPrediction.prediction_score))
            .limit(limit)
        )

        result = await self.session.exec(stmt)
        false_negatives = result.all()

        formatted_results = []

        for prediction, tx in false_negatives:
            formatted_results.append(
                {
                    "transaction_id": str(tx.id),
                    "reference": tx.reference,
                    "amount": str(tx.amount),
                    "prediction_score": prediction.prediction_score,
                    "transaction_date": tx.created_at.isoformat(),
                    "prediction_date": prediction.prediction_timestamp.isoformat(),
                    "features": prediction.input_features,
                    "metadata": tx.transaction_metadata,
                }
            )
        model = await self.session.get(MLModel, model_id)

        if model and model.mlflow_run_id and formatted_results:
            with mlflow.start_run(run_id=model.mlflow_run_id):
                mlflow.log_metric("false_negative_count", len(formatted_results))

                for i, fp in enumerate(formatted_results[:5]):
                    mlflow.log_metric(f"top_fp_{i+1}_score", fp["prediction_score"])

                import json

                fp_path = "/tmp/false_negatives.json"

                with open(fp_path, "w") as f:
                    json.dump(formatted_results, f)

                mlflow.log_artifact(fp_path, "evaluation/false_negatives")
        return formatted_results

    async def compare_models(
        self,
        model_ids: List[UUID],
        start_date: Optional[datetime] = None,
        end_date: Optional[datetime] = None,
    ) -> Dict[str, Any]:

        models_metrics = {}

        with mlflow.start_run(
            experiment_id=self.experiment_id, run_name="model_comparison"
        ) as run:
            comparison_run_id = run.info.run_id

            mlflow.log_param("comparison_date", datetime.now(timezone.utc).isoformat())

            mlflow.log_param("models_compared", len(model_ids))

            for model_id in model_ids:
                try:
                    metrics = await self.evaluate_model_performance(
                        model_id=model_id, start_date=start_date, end_date=end_date
                    )

                    models_metrics[str(model_id)] = {
                        "name": metrics["model_name"],
                        "version": metrics["model_version"],
                        "metrics": metrics["metrics"],
                    }

                    mlflow.log_metrics(
                        {
                            f"model_{model_id}_auc": metrics["metrics"]["auc"],
                            f"model_{model_id}_precision": metrics["metrics"][
                                "precision"
                            ],
                            f"model_{model_id}_recall": metrics["metrics"]["recall"],
                            f"model_{model_id}_f1": metrics["metrics"]["f1"],
                        }
                    )
                except Exception as e:
                    logger.error(f"Error evaluating model {model_id}: {e}")
                    models_metrics[str(model_id)] = {"error": str(e)}

            if len(models_metrics) > 1:
                try:
                    import matplotlib.pyplot as plt
                    import numpy as np

                    metrics_to_plot = ["auc", "precision", "recall", "f1"]

                    model_names = [
                        m["name"] + " " + m["version"]
                        for m in models_metrics.values()
                        if "error" not in m
                    ]

                    plt.figure(figsize=(10, 6))

                    x = np.arange(len(metrics_to_plot))

                    width = 0.8 / len(model_names)

                    for i, (model_id, model_data) in enumerate(models_metrics.items()):
                        if "error" not in model_data:
                            values = [
                                model_data["metrics"].get(m, 0) for m in metrics_to_plot
                            ]
                            plt.bar(
                                x + i * width,
                                values,
                                width,
                                label=model_data["name"] + " " + model_data["version"],
                            )
                    plt.ylabel("Score")
                    plt.title("Model Performance Comparison")

                    plt.xticks(x + width / 2, metrics_to_plot)
                    plt.legend()

                    plt.tight_layout()

                    comparison_plot_path = "/tmp/model_comparison.png"
                    plt.savefig(comparison_plot_path)

                    mlflow.log_artifact(comparison_plot_path, "evaluation/comparison")
                except Exception as e:
                    logger.error(f"Error creating comparison visualization: {e}")
        return {
            "comparison_run_id": comparison_run_id,
            "comparison_period": {
                "start": start_date.isoformat() if start_date else None,
                "end": end_date.isoformat() if end_date else None,
            },
            "models": models_metrics,
        }
