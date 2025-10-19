package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"example.com/transactions/v1/src/db"
	"example.com/transactions/v1/src/models"
	"github.com/go-chi/chi/v5"
)

// HandlerGetTransactionsByAccount fetches transaction history for an account
func HandlerGetTransactionsByAccount(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "accountId")
	if accountID == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	params := &models.TransactionHistoryParams{
		AccountID: accountID,
		Limit:     20,  // default limit
		Offset:    0,   // default offset
	}

	// Get limit and offset from query params
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			params.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			params.Offset = offset
		}
	}

	// Get optional filters
	if txType := r.URL.Query().Get("type"); txType != "" {
		params.TransactionType = models.TransactionType(txType)
	}

	if category := r.URL.Query().Get("category"); category != "" {
		params.TransactionCategory = models.TransactionCategory(category)
	}

	if status := r.URL.Query().Get("status"); status != "" {
		params.Status = models.TransactionStatus(status)
	}

	// Fetch transactions
	transactions, err := db.GetTransactionsByAccountID(accountID, params)
	if err != nil {
		log.Printf("Failed to get transactions for account %s: %v", accountID, err)
		http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transactions)
}

// HandlerGetTransactionsByWallet fetches transaction history for a wallet
func HandlerGetTransactionsByWallet(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletId")
	if walletID == "" {
		http.Error(w, "Wallet ID is required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	params := &models.TransactionHistoryParams{
		WalletID: walletID,
		Limit:    20,  // default limit
		Offset:   0,   // default offset
	}

	// Get limit and offset from query params
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			params.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			params.Offset = offset
		}
	}

	// Get optional filters
	if txType := r.URL.Query().Get("type"); txType != "" {
		params.TransactionType = models.TransactionType(txType)
	}

	if category := r.URL.Query().Get("category"); category != "" {
		params.TransactionCategory = models.TransactionCategory(category)
	}

	if status := r.URL.Query().Get("status"); status != "" {
		params.Status = models.TransactionStatus(status)
	}

	// Fetch transactions
	transactions, err := db.GetTransactionsByWalletID(walletID, params)
	if err != nil {
		log.Printf("Failed to get transactions for wallet %s: %v", walletID, err)
		http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transactions)
}

// HandlerGetTransaction fetches a single transaction by ID
func HandlerGetTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "transactionId")
	if transactionID == "" {
		http.Error(w, "Transaction ID is required", http.StatusBadRequest)
		return
	}

	transaction, err := db.GetTransaction(transactionID)
	if err != nil {
		log.Printf("Failed to get transaction %s: %v", transactionID, err)
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transaction)
}
