package events

import "time"

// VirtualCardCreatedEvent is emitted when a virtual card is created
type VirtualCardCreatedEvent struct {
	CardID         string    `json:"cardId"`
	AccountID      string    `json:"accountId"`
	WalletId       string    `json:"walletId"`
	CardType       string    `json:"cardType"`
	Currency       string    `json:"currency"`
	LastFourDigits string    `json:"lastFourDigits"`
	Timestamp      time.Time `json:"timestamp"`
}

// VirtualCardBlockedEvent is emitted when a virtual card is blocked
type VirtualCardBlockedEvent struct {
	CardID       string    `json:"cardId"`
	AccountID    string    `json:"accountId"`
	BlockReason  string    `json:"blockReason"`
	BlockedBy    string    `json:"blockedBy"`
	Timestamp    time.Time `json:"timestamp"`
}

// VirtualCardToppedUpEvent is emitted when a virtual card is topped up
type VirtualCardToppedUpEvent struct {
	CardID       string    `json:"cardId"`
	AccountID    string    `json:"accountId"`
	Amount       float64   `json:"amount"`
	NewBalance   float64   `json:"newBalance"`
	Timestamp    time.Time `json:"timestamp"`
}

// PhysicalCardRequestedEvent is emitted when a physical card is requested
type PhysicalCardRequestedEvent struct {
	CardID         string    `json:"cardId"`
	AccountID      string    `json:"accountId"`
	DeliveryAddress string   `json:"deliveryAddress"`
	DeliveryCity    string   `json:"deliveryCity"`
	DeliveryCountry string   `json:"deliveryCountry"`
	Timestamp       time.Time `json:"timestamp"`
}

// VirtualCardDeletedEvent is emitted when a virtual card is deleted
type VirtualCardDeletedEvent struct {
	CardID        string    `json:"cardId"`
	AccountID     string    `json:"accountId"`
	LastFourDigits string   `json:"lastFourDigits"`
	Timestamp     time.Time `json:"timestamp"`
}
