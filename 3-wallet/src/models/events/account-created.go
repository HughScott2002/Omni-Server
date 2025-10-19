package events

type AccountCreatedEvent struct {
	AccountId string `json:"accountId"`
	Currency  string `json:"currency"`
	KYCStatus string `json:"kycstatus"`
}
