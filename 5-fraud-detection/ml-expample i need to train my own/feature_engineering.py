from datetime import datetime, timedelta
from typing import Any
from uuid import UUID

import mlflow
import numpy as np
import pandas as pd
from sqlmodel import select
from sqlmodel.ext.asyncio.session import AsyncSession

from backend.app.bank_account.models import BankAccount
from backend.app.core.ai.models import TransactionRiskScore
from backend.app.core.logging import get_logger
from backend.app.transaction.models import Transaction

logger = get_logger()


class FeatureExtractor:

    def __init__(self, session: AsyncSession):
        self.session = session
        self.feature_names = []

    def _extract_time_features(self, transaction: Transaction) -> dict[str, Any]:
        created_at = transaction.created_at
        is_banking_hours = 1 if 9 <= created_at.hour <= 17 else 0
        is_late_night = 1 if created_at.hour <= 5 or created_at.hour >= 23 else 0

        month = created_at.month
        day = created_at.day
        day_of_week = created_at.weekday()
        is_weekend = 1 if day_of_week >= 5 else 0

        is_month_end = 1 if day >= 25 else 0
        is_month_start = 1 if day <= 5 else 0

        return {
            "hour_of_day": created_at.hour,
            "day_of_week": day_of_week,
            "is_weekend": is_weekend,
            "is_banking_hours": is_banking_hours,
            "is_late_night": is_late_night,
            "month": month,
            "day": day,
            "is_month_end": is_month_end,
            "is_month_start": is_month_start,
        }

    def _extract_metadata_features(self, metadata: dict[str, Any]) -> dict[str, Any]:
        features = {}

        if not metadata:
            return features

        currency_value = None

        if "currency" in metadata:
            currency_value = metadata["currency"]
        elif isinstance(metadata.get("from_currency"), str):
            currency_value = metadata["from_currency"]

        if currency_value:
            features["currency"] = currency_value
            features[f"currency_{currency_value}"] = 1

        if "converted_amount" in metadata:
            features["is_currency_conversion"] = 1
            try:
                conversion_ratio = float(metadata.get("converted_amount", 0)) / float(
                    metadata.get("original_amount", 1) or metadata.get("amount", 1)
                )
                features["conversion_ratio"] = conversion_ratio
            except (ValueError, ZeroDivisionError):
                features["conversion_ratio"] = 0
        else:
            features["is_currency_conversion"] = 0
        return features

    async def _extract_account_features(
        self, account_id: UUID, is_sender: bool
    ) -> dict[str, Any]:
        prefix = "sender" if is_sender else "receiver"

        account = await self.session.get(BankAccount, account_id)

        if not account:
            return {f"{prefix}_account_not_found": 1}

        features = {
            f"{prefix}_account_balance": float(account.balance),
            f"{prefix}_account_age_days": (
                datetime.now(account.created_at.tzinfo) - account.created_at
            ).days,
        }

        if hasattr(account, "account_status") and account.account_status:
            features[f"{prefix}_account_status_{account.account_status.value}"] = 1

        if is_sender:
            stmt = select(Transaction).where(
                Transaction.sender_account_id == account_id
            )

        else:
            stmt = select(Transaction).where(
                Transaction.receiver_account_id == account_id
            )

        result = await self.session.exec(stmt)
        transactions = result.all()

        features[f"{prefix}_transaction_count"] = len(transactions)

        if transactions:
            amounts = [float(t.amount) for t in transactions]

            features[f"{prefix}_avg_transaction_amount"] = np.mean(amounts)

            if len(transactions) > 1:
                features[f"{prefix}_std_transaction_amount"] = float(np.std(amounts))
                features[f"{prefix}_max_transaction_amount"] = float(np.max(amounts))
                features[f"{prefix}_min_transaction_amount"] = float(np.min(amounts))
            else:
                features[f"{prefix}_std_transaction_amount"] = 0.0
                features[f"{prefix}_max_transaction_amount"] = amounts[0]
                features[f"{prefix}_min_transaction_amount"] = amounts[0]
        return features

    async def _extract_user_history_features(
        self, user_id: UUID, current_time: datetime
    ) -> dict[str, Any]:

        lookback_period = current_time - timedelta(days=90)

        stmt = select(Transaction).where(
            Transaction.sender_id == user_id,
            Transaction.created_at >= lookback_period,
            Transaction.created_at < current_time,
        )

        result = await self.session.exec(stmt)
        transactions = result.all()

        if not transactions:
            return {
                "user_transaction_count_90d": 0,
                "user_avg_amount_90d": 0,
                "user_max_amount_90d": 0,
                "user_transaction_frequency_daily": 0,
            }

        amounts = [float(t.amount) for t in transactions]

        days_in_history = (current_time - lookback_period).days or 1

        tx_per_day = len(transactions) / days_in_history

        features = {
            "user_transaction_count_90d": len(transactions),
            "user_avg_amount_90d": float(np.mean(amounts)) if amounts else 0,
            "user_max_amount_90d": float(np.max(amounts)) if amounts else 0,
            "user_min_amount_90d": float(np.min(amounts)) if amounts else 0,
            "user_std_amount_90d": float(np.std(amounts)) if len(amounts) > 1 else 0,
            "user_transaction_frequency_daily": tx_per_day,
        }

        tx_types = [t.transaction_type.value for t in transactions]

        for tx_type in set(tx_types):
            count = tx_types.count(tx_type)
            features[f"user_tx_type_{tx_type}_ratio"] = count / len(transactions)

        return features

    async def _extract_velocity_features(
        self, user_id: UUID, current_time: datetime
    ) -> dict[str, Any]:

        time_windows = [
            ("1h", timedelta(hours=1)),
            ("1d", timedelta(days=1)),
            ("7d", timedelta(days=7)),
            ("30d", timedelta(days=30)),
        ]

        features = {}

        for window_name, window_size in time_windows:
            lookback_time = current_time - window_size

            stmt = select(Transaction).where(
                Transaction.sender_id == user_id,
                Transaction.created_at >= lookback_time,
                Transaction.created_at < current_time,
            )

            result = await self.session.exec(stmt)
            transactions = result.all()

            features[f"tx_count_{window_name}"] = len(transactions)

            if transactions:
                total_amount = sum(float(t.amount) for t in transactions)
                features[f"tx_total_amount_{window_name}"] = total_amount

                features[f"tx_avg_amount_{window_name}"] = total_amount / len(
                    transactions
                )
            else:
                features[f"tx_total_amount_{window_name}"] = 0
                features[f"tx_avg_amount_{window_name}"] = 0
        return features

    async def extract_features_for_transaction(
        self, transaction: Transaction, mlflow_run_id: str | None = None
    ) -> dict[str, Any]:
        try:
            features = {
                "amount": float(transaction.amount),
                "transaction_type": transaction.transaction_type.value,
                "transaction_category": transaction.transaction_category.value,
            }
            features[f"tx_type_{transaction.transaction_type.value}"] = 1
            features[f"tx_category_{transaction.transaction_category.value}"] = 1

            features.update(self._extract_time_features(transaction))

            if transaction.sender_account_id:
                account_features = await self._extract_account_features(
                    transaction.sender_account_id, is_sender=True
                )
                features.update(account_features)

            if transaction.receiver_account_id:
                account_features = await self._extract_account_features(
                    transaction.receiver_account_id, is_sender=False
                )
                features.update(account_features)

            if transaction.sender_id:
                user_features = await self._extract_user_history_features(
                    transaction.sender_id, transaction.created_at
                )
                features.update(user_features)

            if transaction.sender_id:
                velocity_features = await self._extract_velocity_features(
                    transaction.sender_id, transaction.created_at
                )
                features.update(velocity_features)

            if transaction.transaction_metadata:
                metadata_features = self._extract_metadata_features(
                    transaction.transaction_metadata
                )
                features.update(metadata_features)

            if mlflow_run_id:
                tx_id_short = str(transaction.id)[-8:]
                try:
                    with mlflow.start_run(run_id=mlflow_run_id):
                        mlflow.log_metric(
                            f"tx_{tx_id_short}_feature_count", len(features)
                        )
                        mlflow.log_metric(
                            f"tx_{tx_id_short}_amount", features.get("amount", 0)
                        )
                        mlflow.log_metric(
                            f"tx_{tx_id_short}_hour", features.get("hour_of_day", 0)
                        )
                        mlflow.log_param(f"processed_tx_{tx_id_short}", 1)

                except Exception as mlflow_error:
                    logger.warning(
                        f"MLflow logging error for transaction {tx_id_short}:{mlflow_error}"
                    )

            return features

        except Exception as e:
            logger.error(f"Error extracting features: {e}")
            return {
                "amount": float(transaction.amount),
                "hour_of_day": transaction.created_at.hour,
                "day_of_week": transaction.created_at.weekday(),
                "error_in_feature_extraction": 1,
            }


async def prepare_training_dataset(
    session: AsyncSession,
    start_date: datetime,
    end_date: datetime,
    mlflow_run_id: str | None = None,
) -> pd.DataFrame:

    logger.info(f"Preparing training dataset from {start_date} to {end_date}")

    stmt = select(Transaction).where(
        Transaction.created_at >= start_date, Transaction.created_at <= end_date
    )

    result = await session.exec(stmt)
    transactions = result.all()

    logger.info(f"Found {len(transactions)} transactions for feature extraction")

    active_run = mlflow.active_run()
    if (
        mlflow_run_id
        and active_run is not None
        and active_run.info.run_id != mlflow_run_id
    ):
        logger.info("Ending existing MLflow run before starting a new one")
        mlflow.end_run()

    feature_extractor = FeatureExtractor(session)

    all_features = []

    for tx in transactions:
        features = await feature_extractor.extract_features_for_transaction(
            tx, mlflow_run_id
        )

        features["transaction_id"] = str(tx.id)

        is_fraud = 0

        if tx.transaction_metadata and "fraud_review" in tx.transaction_metadata:
            if tx.transaction_metadata["fraud_review"].get("is_fraud", False):
                is_fraud = 1

        if tx.ai_review_status and tx.ai_review_status == "CONFIRMED_FRAUD":
            is_fraud = 1

        if is_fraud == 0:
            risk_stmt = select(TransactionRiskScore).where(
                TransactionRiskScore.transaction_id == tx.id,
                TransactionRiskScore.is_confirmed_fraud == True,
            )
            risk_result = await session.exec(risk_stmt)
            risk_score = risk_result.first()

            if risk_score:
                is_fraud = 1

        features["is_fraud"] = is_fraud
        all_features.append(features)

    if not all_features:
        logger.warning("No features extracted, returning empty dataframe")
        return pd.DataFrame()

    df = pd.DataFrame(all_features)

    if mlflow_run_id:
        try:
            active_run = mlflow.active_run()

            in_correct_run = active_run and active_run.info.run_id == mlflow_run_id

            with mlflow.start_run(
                run_id=mlflow_run_id if not in_correct_run else None,
                nested=True if in_correct_run else False,
            ):
                mlflow.log_param("dataset_rows", len(df))
                mlflow.log_param("dataset_columns", len(df.columns))
                mlflow.log_param("fraud_ratio", df["is_fraud"].mean())
                mlflow.log_param("feature_count", len(df.columns) - 2)

                columns_sample = df.columns.tolist()[:50]

                mlflow.log_param("columns_sample", columns_sample)
                mlflow.log_metric("fraud_count", df["is_fraud"].sum())
                mlflow.log_metric("legitimate_count", len(df) - df["is_fraud"].sum())

        except Exception as e:
            logger.error(f"Error logging to MLflow: {e}")

            if mlflow.active_run():
                mlflow.end_run()

    string_columns = []

    for col in df.columns:
        if col not in ["transaction_id", "is_fraud"]:
            if df[col].dtype == object:
                try:
                    pd.to_numeric(df[col])
                except Exception:
                    string_columns.append(col)
            elif any(
                keyword in col for keyword in ["status", "type", "category", "currency"]
            ):
                string_columns.append(col)

    if string_columns:
        df = pd.get_dummies(df, columns=string_columns, drop_first=True)

    for col in df.select_dtypes(include=["object"]).columns:
        if col != "transaction_id":
            try:
                df[col] = df[col].astype(str).astype("category")
            except Exception:
                logger.warning(f"Dropping column {col} as it can't be properly encoded")
                df = df.drop(columns=[col])

    df = df.fillna(0)

    logger.info(f"Prepared dataset with {df.shape[0]} rows, {df.shape[1]} columns")

    return df
