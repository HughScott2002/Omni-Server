package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"example.com/m/v2/src/db"
	"example.com/m/v2/src/events"
	"example.com/m/v2/src/models"
	"github.com/segmentio/kafka-go"
)

func ConsumeAccountDeletionEvents(ctx context.Context) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:               []string{"broker:9092"},
		GroupID:               events.ConsumerGroup,
		Topic:                 events.AccountDeletionRequestedTopic,
		MinBytes:              10e3,
		MaxBytes:              10e6,
		MaxWait:               3 * time.Second,
		ReadBackoffMax:        5 * time.Second,
		HeartbeatInterval:     10 * time.Second,
		SessionTimeout:        30 * time.Second,
		StartOffset:           kafka.FirstOffset,
		WatchPartitionChanges: true,
	})
	defer reader.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if err != context.Canceled {
					log.Printf("Error reading message: %v", err)
				}
				time.Sleep(time.Second)
				continue
			}
			if err := processAccountDeletionMessage(msg); err != nil {
				log.Printf("Error processing deletion message: %v", err)
			}
		}
	}
}

func processAccountDeletionMessage(msg kafka.Message) error {
	type AccountDeletionRequestedEvent struct {
		AccountId         string    `json:"accountId"`
		Email             string    `json:"email"`
		RequestedAt       time.Time `json:"requestedAt"`
		ScheduledDeletion time.Time `json:"scheduledDeletion"`
	}
	var event AccountDeletionRequestedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %v", err)
	}

	wallets, err := db.GetWalletsByAccountId(event.AccountId)
	if err != nil {
		return fmt.Errorf("failed to get wallets: %v", err)
	}

	for _, wallet := range wallets {
		wallet.Status = models.WalletStatusDisabled
		wallet.UpdatedAt = time.Now()
		if err := db.UpdateWallet(wallet); err != nil {
			return fmt.Errorf("failed to disable wallet %s: %v", wallet.WalletId, err)
		}
	}
	return nil
}
