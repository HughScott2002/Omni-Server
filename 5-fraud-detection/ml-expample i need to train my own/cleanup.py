import logging

import mlflow

from backend.app.core.ml.config import ml_settings

logger = logging.getLogger(__name__)


def cleanup_mlflow_runs():
    try:
        mlflow.set_tracking_uri(ml_settings.MLFLOW_TRACKING_URI)

        active_run = mlflow.active_run()

        if active_run:
            run_id = active_run.info.run_id
            logger.warning(f"Found active MLflow run {run_id} at startup, ending it")

            mlflow.end_run()

            logger.info(f"Successfully ended active MLflow run {run_id}")

        client = mlflow.MlflowClient()

        try:
            experiment = mlflow.get_experiment_by_name(
                ml_settings.MLFLOW_EXPERIMENT_NAME
            )

            if experiment:
                experiment_id = experiment.experiment_id

                running_runs = client.search_runs(
                    experiment_ids=[experiment_id],
                    filter_string="attributes.status = 'RUNNING'",
                )

                for run in running_runs:
                    logger.warning(
                        f"Found stale RUNNING run {run.info.run_id}, ending it"
                    )

                    client.set_terminated(run.info.run_id, "FINISHED")

                logger.info(f"Cleaned up {len(running_runs)} stale MLflow runs")
        except Exception as e:
            logger.error(f"Error cleaning up stale runs: {e}")

        logger.info("MLflow run cleanup completed successfully")

    except Exception as e:
        logger.error(f"Failed to cleanup MLflow runs: {e}")
