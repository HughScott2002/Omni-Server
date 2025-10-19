package events

import (
	"time"
)

// TransactionCreatedEvent is published when a new transaction is created
type TransactionCreatedEvent struct {
	TransactionID       string    `json:"transactionId"`
	Reference           string    `json:"reference"`
	SenderAccountID     string    `json:"senderAccountId,omitempty"`
	ReceiverAccountID   string    `json:"receiverAccountId,omitempty"`
	SenderWalletID      string    `json:"senderWalletId,omitempty"`
	ReceiverWalletID    string    `json:"receiverWalletId,omitempty"`
	Amount              float64   `json:"amount"`
	Currency            string    `json:"currency"`
	TransactionType     string    `json:"transactionType"`
	TransactionCategory string    `json:"transactionCategory"`
	Status              string    `json:"status"`
	Description         string    `json:"description"`
	Timestamp           time.Time `json:"timestamp"`
}

// TransactionCompletedEvent is published when a transaction completes successfully
type TransactionCompletedEvent struct {
	TransactionID        string    `json:"transactionId"`
	Reference            string    `json:"reference"`
	SenderAccountID      string    `json:"senderAccountId,omitempty"`
	ReceiverAccountID    string    `json:"receiverAccountId,omitempty"`
	SenderWalletID       string    `json:"senderWalletId,omitempty"`
	ReceiverWalletID     string    `json:"receiverWalletId,omitempty"`
	Amount               float64   `json:"amount"`
	Currency             string    `json:"currency"`
	TransactionType      string    `json:"transactionType"`
	TransactionCategory  string    `json:"transactionCategory"`
	Description          string    `json:"description"`
	SenderBalanceAfter   float64   `json:"senderBalanceAfter,omitempty"`
	ReceiverBalanceAfter float64   `json:"receiverBalanceAfter,omitempty"`
	Timestamp            time.Time `json:"timestamp"`
	CompletedAt          time.Time `json:"completedAt"`
}

// TransactionFailedEvent is published when a transaction fails
type TransactionFailedEvent struct {
	TransactionID       string    `json:"transactionId"`
	Reference           string    `json:"reference"`
	SenderAccountID     string    `json:"senderAccountId,omitempty"`
	ReceiverAccountID   string    `json:"receiverAccountId,omitempty"`
	Amount              float64   `json:"amount"`
	Currency            string    `json:"currency"`
	TransactionType     string    `json:"transactionType"`
	TransactionCategory string    `json:"transactionCategory"`
	Description         string    `json:"description"`
	FailedReason        string    `json:"failedReason"`
	Timestamp           time.Time `json:"timestamp"`
}

// MoneyReceivedEvent is published when an account receives money
type MoneyReceivedEvent struct {
	AccountID       string    `json:"accountId"`
	WalletID        string    `json:"walletId"`
	TransactionID   string    `json:"transactionId"`
	Reference       string    `json:"reference"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	SenderAccountID string    `json:"senderAccountId,omitempty"`
	SenderOmniTag   string    `json:"senderOmniTag,omitempty"`
	Description     string    `json:"description"`
	BalanceAfter    float64   `json:"balanceAfter"`
	Timestamp       time.Time `json:"timestamp"`
}

// MoneySentEvent is published when an account sends money
type MoneySentEvent struct {
	AccountID         string    `json:"accountId"`
	WalletID          string    `json:"walletId"`
	TransactionID     string    `json:"transactionId"`
	Reference         string    `json:"reference"`
	Amount            float64   `json:"amount"`
	Currency          string    `json:"currency"`
	ReceiverAccountID string    `json:"receiverAccountId,omitempty"`
	ReceiverOmniTag   string    `json:"receiverOmniTag,omitempty"`
	Description       string    `json:"description"`
	BalanceAfter      float64   `json:"balanceAfter"`
	Timestamp         time.Time `json:"timestamp"`
}

// CardPurchaseEvent is published when a card is used for purchase
type CardPurchaseEvent struct {
	AccountID        string    `json:"accountId"`
	WalletID         string    `json:"walletId"`
	CardID           string    `json:"cardId"`
	TransactionID    string    `json:"transactionId"`
	Reference        string    `json:"reference"`
	Amount           float64   `json:"amount"`
	Currency         string    `json:"currency"`
	MerchantName     string    `json:"merchantName"`
	MerchantCategory string    `json:"merchantCategory"`
	Description      string    `json:"description"`
	CardBalance      float64   `json:"cardBalance"`
	WalletBalance    float64   `json:"walletBalance"`
	Timestamp        time.Time `json:"timestamp"`
}

// CardRefundEvent is published when a card purchase is refunded
type CardRefundEvent struct {
	AccountID         string    `json:"accountId"`
	WalletID          string    `json:"walletId"`
	CardID            string    `json:"cardId"`
	TransactionID     string    `json:"transactionId"`
	Reference         string    `json:"reference"`
	OriginalReference string    `json:"originalReference"`
	Amount            float64   `json:"amount"`
	Currency          string    `json:"currency"`
	MerchantName      string    `json:"merchantName"`
	Description       string    `json:"description"`
	CardBalance       float64   `json:"cardBalance"`
	WalletBalance     float64   `json:"walletBalance"`
	Timestamp         time.Time `json:"timestamp"`
}
