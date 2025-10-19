package models

import "time"

// RiskAssessmentRequest represents a request to assess transaction risk
type RiskAssessmentRequest struct {
	TransactionID     string                 `json:"transactionId"`
	SenderAccountID   string                 `json:"senderAccountId"`
	ReceiverAccountID string                 `json:"receiverAccountId"`
	Amount            float64                `json:"amount"`
	Currency          string                 `json:"currency"`
	TransactionType   string                 `json:"transactionType"`
	Description       string                 `json:"description,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// RiskAssessmentResponse represents the risk assessment result
type RiskAssessmentResponse struct {
	TransactionID string    `json:"transactionId"`
	RiskScore     float64   `json:"riskScore"` // 0-100, where 0 is no risk, 100 is highest risk
	RiskLevel     string    `json:"riskLevel"` // "low", "medium", "high"
	Decision      string    `json:"decision"`  // "approve", "review", "decline"
	Reasons       []string  `json:"reasons,omitempty"`
	AssessedAt    time.Time `json:"assessedAt"`
}

// Risk levels
const (
	RiskLevelLow    = "low"
	RiskLevelMedium = "medium"
	RiskLevelHigh   = "high"
)

// Decisions
const (
	DecisionApprove = "approve"
	DecisionReview  = "review"
	DecisionDecline = "decline"
)
