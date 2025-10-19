package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"omni/fraud-detection/src/models"
	"omni/fraud-detection/src/utils"
)

// HandlerAssessRisk handles risk assessment requests for transactions
func HandlerAssessRisk(w http.ResponseWriter, r *http.Request) {
	var req models.RiskAssessmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode risk assessment request: %v", err)
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
	log.Printf("Risk assessed for transaction %s: score=%.2f, level=%s, decision=%s",
		response.TransactionID, response.RiskScore, response.RiskLevel, response.Decision)

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
