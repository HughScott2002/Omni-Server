package utils

import "fmt"

const MinPasswordLength = 8

// ValidatePassword enforces the minimum password strength accepted for
// registration and password changes.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	return nil
}
