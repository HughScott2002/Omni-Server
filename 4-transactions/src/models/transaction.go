package models

import (
	"time"
)

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeDeposit          TransactionType = "deposit"
	TransactionTypeWithdrawal       TransactionType = "withdrawal"
	TransactionTypeTransfer         TransactionType = "transfer"
	TransactionTypeCardPurchase     TransactionType = "card_purchase"
	TransactionTypeCardRefund       TransactionType = "card_refund"
	TransactionTypeReversal         TransactionType = "reversal"
	TransactionTypeFeeCharged       TransactionType = "fee_charged"
	TransactionTypeInterestCredited TransactionType = "interest_credited"
)

// TransactionStatus represents the current status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusReversed  TransactionStatus = "reversed"
	TransactionStatusCancelled TransactionStatus = "cancelled"
)

// TransactionCategory represents credit or debit
type TransactionCategory string

const (
	TransactionCategoryCredit TransactionCategory = "credit"
	TransactionCategoryDebit  TransactionCategory = "debit"
)

// Transaction represents a financial transaction in the system
type Transaction struct {
	ID                  string                 `json:"id"`                     // Unique transaction ID
	Reference           string                 `json:"reference"`              // Unique transaction reference for idempotency
	SenderAccountID     string                 `json:"senderAccountId"`        // Account ID of sender (can be empty for deposits)
	ReceiverAccountID   string                 `json:"receiverAccountId"`      // Account ID of receiver (can be empty for withdrawals)
	SenderWalletID      string                 `json:"senderWalletId"`         // Wallet ID of sender
	ReceiverWalletID    string                 `json:"receiverWalletId"`       // Wallet ID of receiver
	CardID              string                 `json:"cardId,omitempty"`       // Card ID if transaction is card-related
	Amount              float64                `json:"amount"`                 // Transaction amount
	Currency            string                 `json:"currency"`               // Currency code (USD, EUR, etc.)
	TransactionType     TransactionType        `json:"transactionType"`        // Type of transaction
	TransactionCategory TransactionCategory    `json:"transactionCategory"`    // Credit or Debit
	Status              TransactionStatus      `json:"status"`                 // Current status
	Description         string                 `json:"description"`            // Transaction description
	BalanceBefore       float64                `json:"balanceBefore"`          // Balance before transaction
	BalanceAfter        float64                `json:"balanceAfter"`           // Balance after transaction
	FailedReason        string                 `json:"failedReason,omitempty"` // Reason if failed
	Metadata            map[string]interface{} `json:"metadata,omitempty"`     // Additional metadata (merchant info, etc.)
	CreatedAt           time.Time              `json:"createdAt"`              // When transaction was created
	CompletedAt         *time.Time             `json:"completedAt,omitempty"`  // When transaction completed
	UpdatedAt           time.Time              `json:"updatedAt"`              // Last update time
}

// TransferRequest represents a request to transfer money between wallets
type TransferRequest struct {
	SenderWalletID  string  `json:"senderWalletId"`  // Wallet ID of sender
	ReceiverOmniTag string  `json:"receiverOmniTag"` // OmniTag of receiver
	Amount          float64 `json:"amount"`          // Amount to transfer
	Description     string  `json:"description"`     // Transfer description
	IdempotencyKey  string  `json:"idempotencyKey"`  // Idempotency key to prevent duplicate transfers
}

// PurchaseRequest represents a card purchase transaction
type PurchaseRequest struct {
	CardID           string  `json:"cardId"`           // Card ID being used
	MerchantName     string  `json:"merchantName"`     // Name of merchant
	MerchantCategory string  `json:"merchantCategory"` // Merchant category (retail, food, etc.)
	Amount           float64 `json:"amount"`           // Purchase amount
	Currency         string  `json:"currency"`         // Currency code
	Description      string  `json:"description"`      // Purchase description
	IdempotencyKey   string  `json:"idempotencyKey"`   // Idempotency key
}

// TransferResponse represents the response to a transfer request
type TransferResponse struct {
	Status          string       `json:"status"`                    // success or failed
	Message         string       `json:"message"`                   // Response message
	TransactionID   string       `json:"transactionId,omitempty"`   // ID of created transaction
	Reference       string       `json:"reference,omitempty"`       // Transaction reference
	SenderBalance   float64      `json:"senderBalance,omitempty"`   // Sender's new balance
	ReceiverBalance float64      `json:"receiverBalance,omitempty"` // Receiver's new balance
	Transaction     *Transaction `json:"transaction,omitempty"`     // Full transaction details
}

// PurchaseResponse represents the response to a purchase request
type PurchaseResponse struct {
	Status        string       `json:"status"`                  // success or failed
	Message       string       `json:"message"`                 // Response message
	TransactionID string       `json:"transactionId,omitempty"` // ID of created transaction
	Reference     string       `json:"reference,omitempty"`     // Transaction reference
	CardBalance   float64      `json:"cardBalance,omitempty"`   // Card's new balance
	WalletBalance float64      `json:"walletBalance,omitempty"` // Wallet's new balance
	Transaction   *Transaction `json:"transaction,omitempty"`   // Full transaction details
}

// TransactionHistoryParams represents parameters for fetching transaction history
type TransactionHistoryParams struct {
	AccountID           string              `json:"accountId"`
	WalletID            string              `json:"walletId,omitempty"`
	StartDate           *time.Time          `json:"startDate,omitempty"`
	EndDate             *time.Time          `json:"endDate,omitempty"`
	TransactionType     TransactionType     `json:"transactionType,omitempty"`
	TransactionCategory TransactionCategory `json:"transactionCategory,omitempty"`
	Status              TransactionStatus   `json:"status,omitempty"`
	MinAmount           float64             `json:"minAmount,omitempty"`
	MaxAmount           float64             `json:"maxAmount,omitempty"`
	Limit               int                 `json:"limit"`
	Offset              int                 `json:"offset"`
}
