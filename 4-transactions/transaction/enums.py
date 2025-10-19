from enum import Enum


class TransactionTypeEnum(str, Enum):
    Deposit = "deposit"
    Withdrawal = "withdrawal"
    Transfer = "transfer"
    Reversal = "reversal"
    Fee_Charged = "fee_charged"
    Loan_Disbursement = "loan_disbursement"
    Loan_Repayment = "loan_repayment"
    Interest_Credited = "interest_credited"


class TransactionStatusEnum(str, Enum):
    Pending = "pending"
    Completed = "completed"
    Failed = "failed"
    Reversed = "reversed"
    Cancelled = "cancelled"


class TransactionCategoryEnum(str, Enum):
    Credit = "credit"
    Debit = "debit"


class TransactionFailureReason(str, Enum):
    INSUFFICIENT_BALANCE = "insufficient_balance"
    INVALID_OTP = "invalid_otp"
    OTP_EXPIRED = "otp_expired"
    CURRENCY_CONVERSION_FAILED = "currency_conversion_failed"
    ACCOUNT_INACTIVE = "account_inactive"
    SYSTEM_ERROR = "system_error"
    INVALID_AMOUNT = "invalid_amount"
    INVALID_ACCOUNT = "invalid_account"
    SELF_TRANSFER = "self_transfer"
    SUSPICIOUS_ACTIVITY = "suspicious_activity"
