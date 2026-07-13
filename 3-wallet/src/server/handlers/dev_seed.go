package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"example.com/m/v2/src/db"
	"example.com/m/v2/src/models"
)

// Dev-only seeding hooks (local environment only). They let the demo seed
// script (scripts/seed-demo-data.py) create wallets and set balances without
// going through Kafka, which is the only production write path.

func devSeedAllowed(w http.ResponseWriter) bool {
	if strings.ToLower(os.Getenv("ENVIRONMENT")) != "local" {
		http.Error(w, "Route not found", http.StatusNotFound)
		return false
	}
	return true
}

// HandlerDevUpsertWallet creates or replaces a wallet from a full Wallet JSON body.
func HandlerDevUpsertWallet(w http.ResponseWriter, r *http.Request) {
	if !devSeedAllowed(w) {
		return
	}

	var wallet models.Wallet
	if err := json.NewDecoder(r.Body).Decode(&wallet); err != nil {
		http.Error(w, "Invalid wallet payload", http.StatusBadRequest)
		return
	}
	if wallet.WalletId == "" || wallet.AccountId == "" {
		http.Error(w, "walletId and accountId are required", http.StatusBadRequest)
		return
	}

	exists, err := db.WalletExists(wallet.WalletId)
	if err == nil && exists {
		err = db.UpdateWallet(&wallet)
	} else {
		err = db.AddWallet(&wallet)
	}
	if err != nil {
		http.Error(w, "Failed to store wallet: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wallet)
}

// HandlerDevSetBalance sets a wallet's balance: {"walletId": "...", "balance": 123.45}
func HandlerDevSetBalance(w http.ResponseWriter, r *http.Request) {
	if !devSeedAllowed(w) {
		return
	}

	var req struct {
		WalletId string  `json:"walletId"`
		Balance  float64 `json:"balance"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.WalletId == "" {
		http.Error(w, "walletId and balance are required", http.StatusBadRequest)
		return
	}

	if err := db.UpdateWalletBalance(req.WalletId, req.Balance); err != nil {
		http.Error(w, "Failed to update balance: "+err.Error(), http.StatusInternalServerError)
		return
	}

	wallet, err := db.GetWallet(req.WalletId)
	if err != nil {
		http.Error(w, "Wallet not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wallet)
}
