package models

import "errors"

// Sentinel errors shared by all database implementations so handlers can map
// them to HTTP status codes with errors.Is.
var (
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrWalletInactive    = errors.New("wallet is not active")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrSameWallet        = errors.New("cannot transfer to the same wallet")
	ErrCurrencyMismatch  = errors.New("wallets have different currencies")
	ErrInvalidAmount     = errors.New("amount must be greater than 0")
)
