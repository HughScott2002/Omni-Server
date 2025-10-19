package models

import "time"

// VirtualCardStatus represents the status of a virtual card
type VirtualCardStatus string

const (
	VirtualCardStatusActive   VirtualCardStatus = "active"
	VirtualCardStatusInactive VirtualCardStatus = "inactive"
	VirtualCardStatusPending  VirtualCardStatus = "pending"
	VirtualCardStatusBlocked  VirtualCardStatus = "blocked"
	VirtualCardStatusExpired  VirtualCardStatus = "expired"
)

// VirtualCardType represents the type of virtual card
type VirtualCardType string

const (
	VirtualCardTypeDebit  VirtualCardType = "debit"
	VirtualCardTypeCredit VirtualCardType = "credit"
)

// VirtualCardBrand represents the card brand
type VirtualCardBrand string

const (
	VirtualCardBrandVisa VirtualCardBrand = "visa"
)

// VirtualCardCurrency represents supported currencies
type VirtualCardCurrency string

const (
	VirtualCardCurrencyUSD VirtualCardCurrency = "USD"
	VirtualCardCurrencyEUR VirtualCardCurrency = "EUR"
	VirtualCardCurrencyGBP VirtualCardCurrency = "GBP"
	VirtualCardCurrencyJMD VirtualCardCurrency = "JMD"
)

// CardBlockReason represents reasons for blocking a card
type CardBlockReason string

const (
	CardBlockReasonLost               CardBlockReason = "lost"
	CardBlockReasonStolen             CardBlockReason = "stolen"
	CardBlockReasonSuspiciousActivity CardBlockReason = "suspicious_activity"
	CardBlockReasonCustomerRequest    CardBlockReason = "customer_request"
)

// VirtualCard represents a virtual card
type VirtualCard struct {
	ID                    string                `json:"id"`
	WalletId              string                `json:"walletId"`
	CardType              VirtualCardType       `json:"cardType"`
	CardBrand             VirtualCardBrand      `json:"cardBrand"`
	Currency              VirtualCardCurrency   `json:"currency"`
	CardStatus            VirtualCardStatus     `json:"cardStatus"`
	DailyLimit            float64               `json:"dailyLimit"`
	MonthlyLimit          float64               `json:"monthlyLimit"`
	NameOnCard            string                `json:"nameOnCard"`
	CardNumber            string                `json:"cardNumber,omitempty"`            // Sensitive - handle carefully
	CVVHash               string                `json:"-"`                               // Never expose in JSON
	ExpiryDate            time.Time             `json:"expiryDate"`
	IsActive              bool                  `json:"isActive"`
	IsPhysicalCardRequest bool                  `json:"isPhysicalCardRequested"`
	BlockReason           *CardBlockReason      `json:"blockReason,omitempty"`
	BlockReasonDesc       *string               `json:"blockReasonDescription,omitempty"`
	AvailableBalance      float64               `json:"availableBalance"`
	TotalToppedUp         float64               `json:"totalToppedUp"`
	LastTopUpDate         *time.Time            `json:"lastTopUpDate,omitempty"`
	BlockedAt             *time.Time            `json:"blockedAt,omitempty"`
	BlockedBy             *string               `json:"blockedBy,omitempty"` // User ID who blocked
	TotalSpendToday       float64               `json:"totalSpendToday"`
	TotalSpentThisMonth   float64               `json:"totalSpentThisMonth"`
	LastTransactionDate   *time.Time            `json:"lastTransactionDate,omitempty"`
	LastTransactionAmount *float64              `json:"lastTransactionAmount,omitempty"`
	PhysicalCardReqAt     *time.Time            `json:"physicalCardRequestedAt,omitempty"`
	DeliveryAddress       *string               `json:"deliveryAddress,omitempty"`
	DeliveryCity          *string               `json:"deliveryCity,omitempty"`
	DeliveryCountry       *string               `json:"deliveryCountry,omitempty"`
	DeliveryPostalCode    *string               `json:"deliveryPostalCode,omitempty"`
	PhysicalCardStatus    *string               `json:"physicalCardStatus,omitempty"`
	CardMetadata          map[string]interface{} `json:"cardMetadata,omitempty"`
	CreatedAt             time.Time             `json:"createdAt"`
	UpdatedAt             time.Time             `json:"updatedAt"`
}

// MaskedCardNumber returns the masked card number
func (v *VirtualCard) MaskedCardNumber() string {
	if v.CardNumber == "" || len(v.CardNumber) < 4 {
		return ""
	}
	return "**** **** **** " + v.CardNumber[len(v.CardNumber)-4:]
}

// LastFourDigits returns the last 4 digits of the card
func (v *VirtualCard) LastFourDigits() string {
	if v.CardNumber == "" || len(v.CardNumber) < 4 {
		return ""
	}
	return v.CardNumber[len(v.CardNumber)-4:]
}

// VirtualCardCreate represents the request to create a virtual card
type VirtualCardCreate struct {
	WalletId     string              `json:"walletId"`
	CardType     VirtualCardType     `json:"cardType"`
	CardBrand    VirtualCardBrand    `json:"cardBrand"`
	Currency     VirtualCardCurrency `json:"currency"`
	DailyLimit   float64             `json:"dailyLimit"`
	MonthlyLimit float64             `json:"monthlyLimit"`
	NameOnCard   string              `json:"nameOnCard"`
}

// VirtualCardUpdate represents the request to update a virtual card
type VirtualCardUpdate struct {
	DailyLimit   *float64 `json:"dailyLimit,omitempty"`
	MonthlyLimit *float64 `json:"monthlyLimit,omitempty"`
	IsActive     *bool    `json:"isActive,omitempty"`
}

// CardBlockRequest represents a request to block a card
type CardBlockRequest struct {
	BlockReason     CardBlockReason `json:"blockReason"`
	BlockReasonDesc string          `json:"blockReasonDescription"`
}

// PhysicalCardRequest represents a request for a physical card
type PhysicalCardRequest struct {
	DeliveryAddress    string `json:"deliveryAddress"`
	DeliveryCity       string `json:"city"`
	DeliveryCountry    string `json:"country"`
	DeliveryPostalCode string `json:"postalCode"`
}

// CardTopUpRequest represents a card top-up request
type CardTopUpRequest struct {
	AccountNumber string  `json:"accountNumber"`
	Amount        float64 `json:"amount"`
	Description   string  `json:"description"`
}

// CardTopUpResponse represents a card top-up response
type CardTopUpResponse struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}
