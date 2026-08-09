package utils

import (
	"log/slog"
	"omni/fraud-detection/src/models"
	"time"
)

// AssessRisk performs real-time rules-based risk assessment on a transaction
func AssessRisk(req models.RiskAssessmentRequest) models.RiskAssessmentResponse {
	// Calculate risk score using rules engine
	riskScore, riskLevel, reasons := CalculateRiskScore(req)

	// Determine decision based on risk score
	decision := DetermineDecision(riskScore, riskLevel)

	// Log assessment
	slog.Info("Risk Assessment - TxID, Score, Level, Decision, Reasons", "transactionId", req.TransactionID, "riskScore", riskScore, "riskLevel", riskLevel, "decision", decision, "len", len(reasons))

	// Store transaction in history for velocity checks (only if not declined immediately)
	if decision != models.DecisionDecline {
		store := GetTransactionStore()
		store.AddTransaction(TransactionHistory{
			TransactionID:     req.TransactionID,
			SenderAccountID:   req.SenderAccountID,
			ReceiverAccountID: req.ReceiverAccountID,
			Amount:            req.Amount,
			Timestamp:         time.Now(),
		})
	}

	return models.RiskAssessmentResponse{
		TransactionID: req.TransactionID,
		RiskScore:     riskScore,
		RiskLevel:     riskLevel,
		Decision:      decision,
		Reasons:       reasons,
		AssessedAt:    time.Now(),
	}
}
