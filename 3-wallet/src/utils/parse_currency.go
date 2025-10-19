package utils

import (
	"fmt"
	"strings"

	"example.com/m/v2/src/models"
)

// Add this function to validate and convert currency strings
func ParseCurrency(c string) (models.Currency, error) {
	switch strings.ToUpper(c) {
	case string(models.CurrencyUSD):
		return models.CurrencyUSD, nil
	case string(models.CurrencyEUR):
		return models.CurrencyEUR, nil
	case string(models.CurrencyGBP):
		return models.CurrencyGBP, nil
	case string(models.CurrencyJMD):
		return models.CurrencyJMD, nil
	case string(models.CurrencyTTD):
		return models.CurrencyTTD, nil
	default:
		return "", fmt.Errorf("unsupported currency: %s", c)
	}
}
