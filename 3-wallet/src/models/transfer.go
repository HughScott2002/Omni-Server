package models

// WalletTransferRequest is the body of POST /api/wallets/transfer.
// Reference is the caller-assigned identity of the transfer: replaying the
// same reference returns the original result instead of moving money twice
// (TigerBeetle-style client-assigned transfer IDs).
type WalletTransferRequest struct {
	FromWalletId string  `json:"fromWalletId"`
	ToWalletId   string  `json:"toWalletId"`
	Amount       float64 `json:"amount"`
	Reference    string  `json:"reference"`
}

// WalletTransferResult carries both wallets as they were the moment the
// transfer applied.
type WalletTransferResult struct {
	FromWallet *Wallet `json:"fromWallet"`
	ToWallet   *Wallet `json:"toWallet"`
}
