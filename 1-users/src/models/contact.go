// 1-users/src/models/contact.go
package models

import "time"

// ContactStatus represents the status of a contact request
type ContactStatus string

const (
	ContactStatusPending  ContactStatus = "pending"
	ContactStatusAccepted ContactStatus = "accepted"
	ContactStatusRejected ContactStatus = "rejected"
	ContactStatusBlocked  ContactStatus = "blocked"
)

// Contact represents a contact/friend relationship between two users
type Contact struct {
	ID            string        `json:"id"`                      // Unique contact ID
	RequesterID   string        `json:"requesterId"`             // User who sent the request
	AddresseeID   string        `json:"addresseeId"`             // User who received the request
	Status        ContactStatus `json:"status"`                  // pending, accepted, rejected, blocked
	RequestedAt   time.Time     `json:"requestedAt"`             // When the request was sent
	RespondedAt   *time.Time    `json:"respondedAt,omitempty"`   // When the request was accepted/rejected
	LastMessageAt *time.Time    `json:"lastMessageAt,omitempty"` // For future messaging feature
}

// ContactInfo represents the information visible about a contact
// Before acceptance: only OmniTag is visible
// After acceptance: full details are visible
type ContactInfo struct {
	AccountID  string        `json:"accountId"`
	OmniTag    string        `json:"omniTag"`
	FirstName  string        `json:"firstName,omitempty"` // Only visible after acceptance
	LastName   string        `json:"lastName,omitempty"`  // Only visible after acceptance
	Email      string        `json:"email,omitempty"`     // Only visible after acceptance
	Status     ContactStatus `json:"status"`              // Status of this contact relationship
	AddedAt    time.Time     `json:"addedAt"`             // When they became contacts
	IsAccepted bool          `json:"isAccepted"`          // Helper field for frontend
}

// ContactRequest represents a pending contact request
type ContactRequest struct {
	ContactID   string    `json:"contactId"`
	FromUser    UserBasic `json:"fromUser"` // Only OmniTag visible
	ToUser      UserBasic `json:"toUser"`   // Only OmniTag visible
	Status      string    `json:"status"`   // pending, accepted, rejected
	RequestedAt time.Time `json:"requestedAt"`
}

// UserBasic represents basic user info (only OmniTag before acceptance)
type UserBasic struct {
	AccountID string `json:"accountId"`
	OmniTag   string `json:"omniTag"`
	FirstName string `json:"firstName,omitempty"` // Only included if accepted
	LastName  string `json:"lastName,omitempty"`  // Only included if accepted
	Email     string `json:"email,omitempty"`     // Only included if accepted
}
