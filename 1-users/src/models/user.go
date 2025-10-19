package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type KYCStatus int
type AccountStatus int

const (
	KYCStatusPending  KYCStatus = iota // 0
	KYCStatusApproved                  // 1
	KYCStatusRejected                  // 2
)
const (
	AccountStatusActive AccountStatus = iota
	AccountStatusDisabled
	AccountStatusPendingDeletion
)

type User struct {
	AccountId           string `json:"accountId"`
	FirstName           string `json:"firstName"`
	LastName            string `json:"lastName"`
	Phone               string `json:"phone"`
	Address             string `json:"address"`
	City                string `json:"city"`
	State               string `json:"state"`
	Country             string `json:"country"`
	Currency            string `json:"currency"`
	PostalCode          string `json:"postalCode"`
	DOB                 string `json:"dob"`
	GovId               string `json:"govId"`
	Email               string `json:"email"`
	OmniTag             string `json:"omniTag"` // Unique user tag (max 5 alphanumeric chars, case-sensitive)
	UnHashedPassword    string `json:"password"`
	HashedPassword      string
	KYCStatus           KYCStatus     `json:"kycstatus"`
	DataAuthorization   bool          `json:"dataAuthorization"`
	Status              AccountStatus `json:"status"`
	DeletionRequestedAt *time.Time    `json:"deletionRequestedAt,omitempty"`
}

func (s KYCStatus) String() string {
	switch s {
	case KYCStatusPending:
		return "pending"
	case KYCStatusApproved:
		return "approved"
	case KYCStatusRejected:
		return "rejected"
	default:
		return "unknown"
	}
}
func (s AccountStatus) String() string {
	switch s {
	case AccountStatusActive:
		return "active"
	case AccountStatusDisabled:
		return "disabled"
	case AccountStatusPendingDeletion:
		return "pending_deletion"
	default:
		return "unknown"
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (s *KYCStatus) UnmarshalJSON(data []byte) error {
	var status string
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}

	switch status {
	case "pending":
		*s = KYCStatusPending
	case "approved":
		*s = KYCStatusApproved
	case "rejected":
		*s = KYCStatusRejected
	default:
		return fmt.Errorf("invalid KYC status: %s", status)
	}

	return nil
}
func (s *AccountStatus) UnmarshalJSON(data []byte) error {
	var status string
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}

	switch status {
	case "active":
		*s = AccountStatusActive
	case "disabled":
		*s = AccountStatusDisabled
	case "pending_deletion":
		*s = AccountStatusPendingDeletion
	default:
		return fmt.Errorf("invalid account status: %s", status)
	}
	return nil
}

func (s AccountStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (u *User) IsActive() bool {
	return u.Status == AccountStatusActive
}

// MarshalJSON implements the json.Marshaler interface
func (s KYCStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (u *User) CanLogin() bool {
	return u.Status == AccountStatusActive
}

func (u *User) RequestDeletion() error {
	if u.Status != AccountStatusActive {
		return fmt.Errorf("account must be active to request deletion")
	}
	now := time.Now()
	u.Status = AccountStatusPendingDeletion
	u.DeletionRequestedAt = &now
	return nil
}
