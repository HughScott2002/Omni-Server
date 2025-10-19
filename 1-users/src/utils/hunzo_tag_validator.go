package utils

import (
	"fmt"
	"regexp"
)

// ValidateOmniTag validates that a Omni Tag meets the requirements:
// - Only alphanumeric characters (letters a-z, A-Z and digits 0-9)
// - Maximum 5 characters
// - Case sensitive
// - At least 1 character
func ValidateOmniTag(tag string) error {
	if tag == "" {
		return fmt.Errorf("Omni tag cannot be empty")
	}

	if len(tag) > 5 {
		return fmt.Errorf("Omni tag must be 5 characters or less, got %d characters", len(tag))
	}

	// Only allow alphanumeric: a-z, A-Z, 0-9
	matched, err := regexp.MatchString("^[a-zA-Z0-9]+$", tag)
	if err != nil {
		return fmt.Errorf("error validating Omni tag: %v", err)
	}

	if !matched {
		return fmt.Errorf("Omni tag must only contain letters (a-z, A-Z) and numbers (0-9)")
	}

	return nil
}
