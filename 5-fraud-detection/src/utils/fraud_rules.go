package utils

import (
	"fmt"
	"math"
	"strings"
	"time"

	"omni/fraud-detection/src/models"
)

// FraudRule represents a single fraud detection rule
type FraudRule struct {
	Name        string
	Description string
	RiskPoints  float64
	Check       func(models.RiskAssessmentRequest) (bool, string)
}

// Risk thresholds
const (
	HighRiskThreshold   = 50.0
	MediumRiskThreshold = 25.0
	DeclineThreshold    = 70.0
	ReviewThreshold     = 50.0
)

// Amount thresholds
const (
	VeryLargeAmount    = 10000.0
	LargeAmount        = 5000.0
	ModerateAmount     = 1000.0
	SuspiciousMaxAmount = 9999.99
	SuspiciousMinAmount = 9990.0
)

// Velocity thresholds
const (
	MaxTransactionsPerHour   = 10
	MaxTransactionsPerDay    = 50
	MaxAmountPerHour         = 5000.0
	MaxAmountPerDay          = 20000.0
	MaxRepeatTransactions    = 5
)

// GetFraudRules returns all fraud detection rules
func GetFraudRules() []FraudRule {
	return []FraudRule{
		// Amount-based rules
		{
			Name:        "Very Large Transaction",
			Description: "Transaction amount exceeds $10,000",
			RiskPoints:  30.0,
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				if req.Amount > VeryLargeAmount {
					return true, fmt.Sprintf("Very large amount: $%.2f", req.Amount)
				}
				return false, ""
			},
		},
		{
			Name:        "Large Transaction",
			Description: "Transaction amount exceeds $5,000",
			RiskPoints:  15.0,
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				if req.Amount > LargeAmount && req.Amount <= VeryLargeAmount {
					return true, fmt.Sprintf("Large amount: $%.2f", req.Amount)
				}
				return false, ""
			},
		},
		{
			Name:        "Suspicious Amount Pattern",
			Description: "Amount is just below $10,000 (possible structuring)",
			RiskPoints:  20.0,
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				if req.Amount >= SuspiciousMinAmount && req.Amount <= SuspiciousMaxAmount {
					return true, fmt.Sprintf("Suspicious amount pattern: $%.2f (possible structuring)", req.Amount)
				}
				return false, ""
			},
		},
		{
			Name:        "Round Amount",
			Description: "Exact round number (e.g., $1000.00)",
			RiskPoints:  5.0,
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				if req.Amount >= ModerateAmount && req.Amount == math.Floor(req.Amount) {
					return true, fmt.Sprintf("Round amount: $%.2f", req.Amount)
				}
				return false, ""
			},
		},
		{
			Name:        "Tiny Transaction",
			Description: "Very small transaction (possible testing)",
			RiskPoints:  8.0,
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				if req.Amount < 1.0 && req.Amount > 0 {
					return true, fmt.Sprintf("Tiny test transaction: $%.2f", req.Amount)
				}
				return false, ""
			},
		},

		// Velocity-based rules
		{
			Name:        "High Transaction Frequency (1 hour)",
			Description: "Too many transactions in the last hour",
			RiskPoints:  25.0,
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				store := GetTransactionStore()
				count := store.CountTransactionsBySender(req.SenderAccountID, 1*time.Hour)
				if count >= MaxTransactionsPerHour {
					return true, fmt.Sprintf("High frequency: %d transactions in last hour", count)
				}
				return false, ""
			},
		},
		{
			Name:        "High Transaction Frequency (24 hours)",
			Description: "Too many transactions in the last day",
			RiskPoints:  15.0,
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				store := GetTransactionStore()
				count := store.CountTransactionsBySender(req.SenderAccountID, 24*time.Hour)
				if count >= MaxTransactionsPerDay {
					return true, fmt.Sprintf("High frequency: %d transactions in last 24 hours", count)
				}
				return false, ""
			},
		},
		{
			Name:        "High Volume (1 hour)",
			Description: "Total amount sent in last hour exceeds limit",
			RiskPoints:  30.0,
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				store := GetTransactionStore()
				total := store.GetTotalAmountBySender(req.SenderAccountID, 1*time.Hour)
				if total+req.Amount > MaxAmountPerHour {
					return true, fmt.Sprintf("High volume: $%.2f sent in last hour", total)
				}
				return false, ""
			},
		},
		{
			Name:        "High Volume (24 hours)",
			Description: "Total amount sent in last day exceeds limit",
			RiskPoints:  20.0,
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				store := GetTransactionStore()
				total := store.GetTotalAmountBySender(req.SenderAccountID, 24*time.Hour)
				if total+req.Amount > MaxAmountPerDay {
					return true, fmt.Sprintf("High volume: $%.2f sent in last 24 hours", total)
				}
				return false, ""
			},
		},
		{
			Name:        "Repeated Transactions to Same Receiver",
			Description: "Multiple transactions to same receiver in short time",
			RiskPoints:  12.0,
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				store := GetTransactionStore()
				txs := store.GetRecentTransactionsBetween(req.SenderAccountID, req.ReceiverAccountID, 1*time.Hour)
				if len(txs) >= MaxRepeatTransactions {
					return true, fmt.Sprintf("Repeated transactions: %d transactions to same receiver in last hour", len(txs))
				}
				return false, ""
			},
		},

		// Pattern-based rules
		{
			Name:        "Suspicious Description Keywords",
			Description: "Description contains suspicious keywords",
			RiskPoints:  15.0,
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				suspiciousKeywords := []string{
					"urgent", "emergency", "cash out", "withdraw all",
					"bitcoin", "crypto", "lottery", "prize", "winner",
					"tax refund", "irs", "government", "inheritance",
					"lawyer", "attorney", "court", "legal fees",
				}

				desc := strings.ToLower(req.Description)
				for _, keyword := range suspiciousKeywords {
					if strings.Contains(desc, keyword) {
						return true, fmt.Sprintf("Suspicious keyword in description: '%s'", keyword)
					}
				}
				return false, ""
			},
		},
		{
			Name:        "Empty Description for Large Amount",
			Description: "No description provided for large transaction",
			RiskPoints:  10.0,
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				if req.Amount > ModerateAmount && strings.TrimSpace(req.Description) == "" {
					return true, "No description for large transaction"
				}
				return false, ""
			},
		},

		// Time-based rules
		{
			Name:        "Late Night Transaction",
			Description: "Transaction during unusual hours (12 AM - 5 AM)",
			RiskPoints:  8.0,
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				hour := time.Now().Hour()
				if hour >= 0 && hour < 5 {
					return true, fmt.Sprintf("Late night transaction at %d:00", hour)
				}
				return false, ""
			},
		},

		// Account-based rules
		{
			Name:        "Same Sender and Receiver",
			Description: "Sender and receiver are the same (should be blocked earlier)",
			RiskPoints:  100.0, // Auto-decline
			Check: func(req models.RiskAssessmentRequest) (bool, string) {
				if req.SenderAccountID == req.ReceiverAccountID {
					return true, "Sender and receiver are the same account"
				}
				return false, ""
			},
		},
	}
}

// CalculateRiskScore evaluates all rules and calculates total risk score
func CalculateRiskScore(req models.RiskAssessmentRequest) (float64, string, []string) {
	rules := GetFraudRules()
	totalRiskScore := 0.0
	reasons := make([]string, 0)
	triggeredRules := 0

	for _, rule := range rules {
		triggered, reason := rule.Check(req)
		if triggered {
			totalRiskScore += rule.RiskPoints
			reasons = append(reasons, reason)
			triggeredRules++
		}
	}

	// Determine risk level
	var riskLevel string
	if totalRiskScore >= HighRiskThreshold {
		riskLevel = models.RiskLevelHigh
	} else if totalRiskScore >= MediumRiskThreshold {
		riskLevel = models.RiskLevelMedium
	} else {
		riskLevel = models.RiskLevelLow
	}

	// If no rules triggered, it's low risk
	if triggeredRules == 0 {
		reasons = append(reasons, "Transaction within normal parameters")
	}

	// Cap risk score at 100
	if totalRiskScore > 100.0 {
		totalRiskScore = 100.0
	}

	return totalRiskScore, riskLevel, reasons
}

// DetermineDecision determines whether to approve, review, or decline
func DetermineDecision(riskScore float64, riskLevel string) string {
	if riskScore >= DeclineThreshold {
		return models.DecisionDecline
	} else if riskScore >= ReviewThreshold {
		return models.DecisionReview
	}
	return models.DecisionApprove
}
