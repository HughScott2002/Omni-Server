package utils

import (
	"crypto/rand"
	"math/big"
	"time"

	"golang.org/x/crypto/argon2"
)

// GenerateVisaCardNumber generates a valid Visa card number using Luhn algorithm
func GenerateVisaCardNumber() (string, error) {
	prefix := "4"

	// Generate 14 random digits
	partialNumber := prefix
	for i := 0; i < 14; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		partialNumber += n.String()
	}

	// Calculate Luhn check digit
	total := 0
	for i := len(partialNumber) - 1; i >= 0; i-- {
		digit := int(partialNumber[i] - '0')
		if (len(partialNumber)-1-i)%2 == 1 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		total += digit
	}

	checkDigit := (10 - (total % 10)) % 10
	return partialNumber + string(rune('0'+checkDigit)), nil
}

// GenerateCVV generates a 3-digit CVV and its hash
func GenerateCVV() (string, string, error) {
	cvv := ""
	for i := 0; i < 3; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", "", err
		}
		cvv += n.String()
	}

	// Hash CVV using Argon2
	cvvHash := hashCVV(cvv)

	return cvv, cvvHash, nil
}

// VerifyCVV verifies a CVV against its hash
func VerifyCVV(cvv, cvvHash string) bool {
	testHash := hashCVV(cvv)
	return testHash == cvvHash
}

// hashCVV hashes a CVV using Argon2
func hashCVV(cvv string) string {
	salt := []byte("omni-card-salt") // In production, use a unique salt per card
	hash := argon2.IDKey([]byte(cvv), salt, 1, 64*1024, 4, 32)
	return string(hash)
}

// GenerateCardExpiryDate generates an expiry date 3 years from now
func GenerateCardExpiryDate() time.Time {
	currentDate := time.Now()
	expiryDate := currentDate.AddDate(3, 0, 0) // Add 3 years

	// Set to last day of the month
	year := expiryDate.Year()
	month := expiryDate.Month()

	// Move to next month, then back one day to get last day of current month
	if month == 12 {
		expiryDate = time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	} else {
		expiryDate = time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	}

	expiryDate = expiryDate.AddDate(0, 0, -1)

	return expiryDate
}
