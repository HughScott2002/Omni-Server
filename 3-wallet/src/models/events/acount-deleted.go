package events

import "time"

type AccountDeletionRequestedEvent struct {
	AccountId         string    `json:"accountId"`
	Email             string    `json:"email"`
	RequestedAt       time.Time `json:"requestedAt"`
	ScheduledDeletion time.Time `json:"scheduledDeletion"`
}
