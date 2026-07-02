package events

type KYCApprovedEvent struct {
	AccountId string `json:"accountId"`
	KYCStatus string `json:"kycstatus"`
}
