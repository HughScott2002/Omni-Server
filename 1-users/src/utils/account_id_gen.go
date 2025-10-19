package utils

import (
	"github.com/google/uuid"
)

// GenerateAccountId creates a new UUID for use as an account ID
func GenerateAccountId() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
