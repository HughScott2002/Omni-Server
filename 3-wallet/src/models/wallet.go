package models

import (
	"time"
)

// WalletStatus represents the current status of a wallet
type WalletStatus string

const (
	WalletStatusActive    WalletStatus = "active"
	WalletStatusInactive  WalletStatus = "inactive"
	WalletStatusSuspended WalletStatus = "suspended"
	WalletStatusDisabled  WalletStatus = "disabled"
)

// WalletType represents the type of wallet
type WalletType string

const (
	WalletTypePrimary WalletType = "PRIMARY"
	WalletTypeSavings WalletType = "SAVINGS"
	WalletTypeEscrow  WalletType = "ESCROW"
)

// Currency represents supported currencies
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
	CurrencyGBP Currency = "GBP"
	CurrencyJMD Currency = "JMD"
	CurrencyTTD Currency = "TTD"
)

// Wallet represents a user's wallet in the system
type Wallet struct {
	WalletId     string       `json:"walletId"`  // Id for the wallet
	AccountId    string       `json:"accountId"` // Id for the account it belongs too
	Type         WalletType   `json:"type"`      // Type of wallet, savings, primary,
	Balance      float64      `json:"balance"`   // Amount in the wallet
	Currency     Currency     `json:"currency"`
	Status       WalletStatus `json:"status"` //active, invactive
	IsDefault    bool         `json:"isDefault"`
	DailyLimit   float64      `json:"dailyLimit"`
	MonthlyLimit float64      `json:"monthlyLimit"`
	LastActivity *time.Time   `json:"lastActivity"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
	// Metadata     interface{}  `json:"metadata,omitempty"`
}

// Transaction represents a wallet transaction
// type Transaction struct {
// 	ID          string    `json:"id"`
// 	WalletID    string    `json:"walletId"`
// 	Type        string    `json:"type"` // credit, debit
// 	Amount      float64   `json:"amount"`
// 	Balance     float64   `json:"balance"` // Balance after transaction
// 	Description string    `json:"description"`
// 	Reference   string    `json:"reference"`
// 	Status      string    `json:"status"` // pending, completed, failed, reversed
// 	CreatedAt   time.Time `json:"createdAt"`
// 	UpdatedAt   time.Time `json:"updatedAt"`
// }

// Balance represents detailed balance information

// type Balance struct {
// 	Available float64 `json:"available"`
// 	Pending   float64 `json:"pending"`
// 	Reserved  float64 `json:"reserved"`
// 	Total     float64 `json:"total"`
// }
