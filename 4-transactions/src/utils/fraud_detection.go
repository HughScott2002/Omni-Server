package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// RiskAssessmentRequest represents the request to the fraud detection service
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

// RiskAssessmentResponse represents the response from the fraud detection service
type RiskAssessmentResponse struct {
	TransactionID string    `json:"transactionId"`
	RiskScore     float64   `json:"riskScore"` // 0-100
	RiskLevel     string    `json:"riskLevel"` // "low", "medium", "high"
	Decision      string    `json:"decision"`  // "approve", "review", "decline"
	Reasons       []string  `json:"reasons,omitempty"`
	AssessedAt    time.Time `json:"assessedAt"`
}

// AssessTransactionRisk calls the fraud detection service to assess a transaction
func AssessTransactionRisk(req RiskAssessmentRequest) (*RiskAssessmentResponse, error) {
	fraudDetectionURL := os.Getenv("FRAUD_DETECTION_URL")
	if fraudDetectionURL == "" {
		fraudDetectionURL = "http://fraud-detection-service:8085"
	}

	// Prepare request body
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal risk assessment request: %w", err)
	}

	// Make HTTP request
	endpoint := fmt.Sprintf("%s/api/fraud-detection/assess", fraudDetectionURL)
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call fraud detection service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fraud detection service returned status %d", resp.StatusCode)
	}

	// Parse response
	var riskResponse RiskAssessmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&riskResponse); err != nil {
		return nil, fmt.Errorf("failed to decode fraud detection response: %w", err)
	}

	log.Printf("Risk assessment for transaction %s: score=%.2f, level=%s, decision=%s",
		riskResponse.TransactionID, riskResponse.RiskScore, riskResponse.RiskLevel, riskResponse.Decision)

	return &riskResponse, nil
}

// IsTransactionApproved checks if the risk assessment approves the transaction
func IsTransactionApproved(assessment *RiskAssessmentResponse) bool {
	return assessment != nil && assessment.Decision == "approve"
}
