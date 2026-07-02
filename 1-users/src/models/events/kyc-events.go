package events

import "omni/src/models"

type KYCApprovedEvent struct {
	AccountId string           `json:"accountId"`
	KYCStatus models.KYCStatus `json:"kycstatus"`
}
