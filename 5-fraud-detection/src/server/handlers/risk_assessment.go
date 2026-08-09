package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"omni/fraud-detection/src/models"
	"omni/fraud-detection/src/utils"
)

// HandlerAssessRisk handles risk assessment requests for transactions
func HandlerAssessRisk(w http.ResponseWriter, r *http.Request) {
	var req models.RiskAssessmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Failed to decode risk assessment request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.TransactionID == "" {
		http.Error(w, "Transaction ID is required", http.StatusBadRequest)
		return
	}

	if req.SenderAccountID == "" {
		http.Error(w, "Sender account ID is required", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		http.Error(w, "Amount must be greater than 0", http.StatusBadRequest)
		return
	}

	// Perform risk assessment
	response := utils.AssessRisk(req)

	// Log the assessment
	slog.Info("Risk assessed for transaction: score=, level=, decision=", "transactionId", response.TransactionID, "riskScore", response.RiskScore, "riskLevel", response.RiskLevel, "decision", response.Decision)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandlerHealth handles health check requests
func HandlerHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "fraud-detection",
	})
}
