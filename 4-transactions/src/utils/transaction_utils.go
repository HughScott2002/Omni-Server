package utils

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GenerateTransactionID generates a unique transaction ID
func GenerateTransactionID() string {
	return uuid.New().String()
}

// GenerateTransactionReference generates a unique transaction reference
// Format: TXN-YYYYMMDD-UUID
func GenerateTransactionReference() string {
	now := time.Now()
	dateStr := now.Format("20060102")
	uniqueID := uuid.New().String()[:8]
	return fmt.Sprintf("TXN-%s-%s", dateStr, uniqueID)
}

// ValidateAmount validates that the amount is greater than 0
func ValidateAmount(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	return nil
}

// ValidateCurrency validates that the currency code is valid
func ValidateCurrency(currency string) error {
	validCurrencies := map[string]bool{
		"USD": true,
		"EUR": true,
		"GBP": true,
		"JMD": true,
		"TTD": true,
	}

	if !validCurrencies[currency] {
		return fmt.Errorf("invalid currency: %s", currency)
	}

	return nil
}

// ValidateIdempotencyKey validates that the idempotency key is not empty
func ValidateIdempotencyKey(key string) error {
	if key == "" {
		return fmt.Errorf("idempotency key is required")
	}
	return nil
}
