package utils

import (
	"omni/fraud-detection/src/models"
	"time"
)

// AssessRisk performs a simple risk assessment on a transaction
// For now, this always approves transactions with a low risk score
// In production, this would use ML models, rule engines, and historical data
func AssessRisk(req models.RiskAssessmentRequest) models.RiskAssessmentResponse {
	// For now, everything is low risk and approved
	// Future enhancements:
	// - Check transaction velocity (how many transactions in last hour/day)
	// - Check unusual amounts for this user
	// - Check geographic location patterns
	// - ML model predictions
	// - Blacklist/whitelist checks
	// - AML screening

	riskScore := 5.0 // Very low risk (0-100 scale)
	reasons := []string{"Transaction within normal parameters"}

	return models.RiskAssessmentResponse{
		TransactionID: req.TransactionID,
		RiskScore:     riskScore,
		RiskLevel:     models.RiskLevelLow,
		Decision:      models.DecisionApprove,
		Reasons:       reasons,
		AssessedAt:    time.Now(),
	}
}
