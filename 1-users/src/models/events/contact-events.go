package events

import "time"

// ContactRequestSentEvent is emitted when a contact request is sent
type ContactRequestSentEvent struct {
	ContactID   string    `json:"contactId"`
	RequesterID string    `json:"requesterId"`
	AddresseeID string    `json:"addresseeId"`
	OmniTag     string    `json:"omniTag"`
	Timestamp   time.Time `json:"timestamp"`
}

// ContactRequestAcceptedEvent is emitted when a contact request is accepted
type ContactRequestAcceptedEvent struct {
	ContactID   string    `json:"contactId"`
	RequesterID string    `json:"requesterId"`
	AddresseeID string    `json:"addresseeId"`
	AcceptedBy  string    `json:"acceptedBy"` // AccountID of person who accepted
	Timestamp   time.Time `json:"timestamp"`
}

// ContactRequestRejectedEvent is emitted when a contact request is rejected
type ContactRequestRejectedEvent struct {
	ContactID   string    `json:"contactId"`
	RequesterID string    `json:"requesterId"`
	AddresseeID string    `json:"addresseeId"`
	RejectedBy  string    `json:"rejectedBy"` // AccountID of person who rejected
	Timestamp   time.Time `json:"timestamp"`
}

// ContactBlockedEvent is emitted when a contact is blocked
type ContactBlockedEvent struct {
	ContactID   string    `json:"contactId"`
	RequesterID string    `json:"requesterId"`
	AddresseeID string    `json:"addresseeId"`
	BlockedBy   string    `json:"blockedBy"` // AccountID of person who blocked
	Timestamp   time.Time `json:"timestamp"`
}
