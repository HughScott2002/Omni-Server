package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"example.com/m/v2/src/db"
	"example.com/m/v2/src/models"
)

// HandlerWalletTransfer moves money between two wallets atomically: both legs
// apply or neither does. Debit and credit are internal to the wallet service;
// this is the only way money moves between wallets.
// POST /api/wallets/transfer
func HandlerWalletTransfer(w http.ResponseWriter, r *http.Request) {
	var req models.WalletTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FromWalletId == "" || req.ToWalletId == "" {
		http.Error(w, "fromWalletId and toWalletId are required", http.StatusBadRequest)
		return
	}
	if req.Reference == "" {
		http.Error(w, "reference is required", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		http.Error(w, models.ErrInvalidAmount.Error(), http.StatusBadRequest)
		return
	}

	result, err := db.Transfer(req.FromWalletId, req.ToWalletId, req.Amount, req.Reference)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrWalletNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, models.ErrWalletInactive),
			errors.Is(err, models.ErrInsufficientFunds),
			errors.Is(err, models.ErrSameWallet),
			errors.Is(err, models.ErrCurrencyMismatch),
			errors.Is(err, models.ErrInvalidAmount):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			slog.Error("Transfer -> (ref) failed", "fromWalletId", req.FromWalletId, "toWalletId", req.ToWalletId, "reference", req.Reference, "error", err)
			http.Error(w, "Failed to execute transfer", http.StatusInternalServerError)
		}
		return
	}

	slog.Info("Transfer from to (ref)", "amount", req.Amount, "fromWalletId", req.FromWalletId, "toWalletId", req.ToWalletId, "reference", req.Reference)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
