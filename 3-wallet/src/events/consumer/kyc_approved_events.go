package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

	slog.Info("Started consuming topic with group", "kYCApprovedTopic", events.KYCApprovedTopic, "consumerGroup", events.ConsumerGroup)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if err != context.Canceled {
					slog.Error("Error reading message", "error", err)
				}
				time.Sleep(time.Second)
				continue
			}

			if err := processKYCApprovedMessage(msg); err != nil {
				slog.Error("Error processing kyc-approved message", "error", err)
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
		slog.Info("Ignoring kyc event for acc# with status", "accountId", event.AccountId, "kYCStatus", event.KYCStatus)
		return nil
	}

	slog.Info("KYC approved for acc#, activating wallets", "accountId", event.AccountId)
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
		slog.Info("Activated wallet for acc", "walletId", wallet.WalletId, "accountId", accountId)
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
		slog.Info("Activated virtual card for acc", "cardId", card.ID, "accountId", accountId)
	}

	return nil
}
