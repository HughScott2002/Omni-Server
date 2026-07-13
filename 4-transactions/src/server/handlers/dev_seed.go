package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"example.com/transactions/v1/src/db"
	"example.com/transactions/v1/src/models"
)

// HandlerDevSeedTransactions bulk-inserts fully-formed transactions, preserving
// their createdAt timestamps. Dev-only hook for scripts/seed-demo-data.py so
// local dashboards have historical data; 404s outside ENVIRONMENT=local.
func HandlerDevSeedTransactions(w http.ResponseWriter, r *http.Request) {
	if strings.ToLower(os.Getenv("ENVIRONMENT")) != "local" {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	var txs []models.Transaction
	if err := json.NewDecoder(r.Body).Decode(&txs); err != nil {
		http.Error(w, "Expected a JSON array of transactions", http.StatusBadRequest)
		return
	}

	seeded := 0
	for i := range txs {
		tx := txs[i]
		if tx.ID == "" || tx.CreatedAt.IsZero() {
			continue
		}
		if err := db.CreateTransaction(&tx); err != nil {
			continue
		}
		seeded++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"seeded": seeded, "received": len(txs)})
}
