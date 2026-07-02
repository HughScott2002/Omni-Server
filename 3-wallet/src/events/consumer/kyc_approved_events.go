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
	eventsModel "example.com/m/v2/src/models/events"
	"github.com/segmentio/kafka-go"
)

// ConsumeKYCApprovedEvents activates an account's wallets (and their pending
// virtual cards) when the users service reports KYC approval. Today that
// approval is auto-issued ~1 minute after registration; a real admin-driven
// KYC service will produce the same event later.
func ConsumeKYCApprovedEvents(ctx context.Context) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:           []string{"broker:9092"},
		GroupID:           events.ConsumerGroup,
		Topic:             events.KYCApprovedTopic,
		MinBytes:          10e3,
		MaxBytes:          10e6,
		MaxWait:           3 * time.Second,
		ReadBackoffMax:    5 * time.Second,
		HeartbeatInterval: 10 * time.Second,
		SessionTimeout:    30 * time.Second,
		StartOffset:       kafka.FirstOffset,

		WatchPartitionChanges: true,
	})
	defer reader.Close()

	log.Printf("Started consuming topic: %s with group: %s", events.KYCApprovedTopic, events.ConsumerGroup)

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

			if err := processKYCApprovedMessage(msg); err != nil {
				log.Printf("Error processing kyc-approved message: %v", err)
				continue
			}
		}
	}
}

func processKYCApprovedMessage(msg kafka.Message) error {
	var event eventsModel.KYCApprovedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal kyc-approved event: %v", err)
	}

	if event.KYCStatus != "approved" {
		log.Printf("Ignoring kyc event for acc#%s with status %q", event.AccountId, event.KYCStatus)
		return nil
	}

	log.Printf("KYC approved for acc#%s, activating wallets", event.AccountId)
	return activateAccountWallets(event.AccountId)
}

func activateAccountWallets(accountId string) error {
	wallets, err := db.GetWalletsByAccountId(accountId)
	if err != nil {
		return fmt.Errorf("failed to list wallets for account %s: %v", accountId, err)
	}

	for _, wallet := range wallets {
		if wallet.Status == models.WalletStatusActive {
			continue
		}
		wallet.Status = models.WalletStatusActive
		wallet.UpdatedAt = time.Now()
		if err := db.UpdateWallet(wallet); err != nil {
			return fmt.Errorf("failed to activate wallet %s: %v", wallet.WalletId, err)
		}
		log.Printf("Activated wallet %s for acc#%s", wallet.WalletId, accountId)
	}

	cards, err := db.GetVirtualCardsByAccountId(accountId)
	if err != nil {
		return fmt.Errorf("failed to list cards for account %s: %v", accountId, err)
	}

	for _, card := range cards {
		if card.CardStatus != models.VirtualCardStatusPending {
			continue
		}
		card.CardStatus = models.VirtualCardStatusActive
		card.IsActive = true
		card.UpdatedAt = time.Now()
		if err := db.UpdateVirtualCard(card); err != nil {
			return fmt.Errorf("failed to activate card %s: %v", card.ID, err)
		}
		log.Printf("Activated virtual card %s for acc#%s", card.ID, accountId)
	}

	return nil
}
