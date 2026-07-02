package consumer

import (
	"encoding/json"
	"testing"

	"example.com/m/v2/src/db"
	"example.com/m/v2/src/models"
	"github.com/segmentio/kafka-go"
)

func setupConsumerTestDB(t *testing.T) {
	t.Helper()
	t.Setenv("ENVIRONMENT", "local")
	t.Setenv("MODE", "memcached")
	if err := db.Init(); err != nil {
		t.Fatalf("db.Init: %v", err)
	}
}

func seedInactiveWallet(t *testing.T, walletId, accountId string) {
	t.Helper()
	err := db.AddWallet(&models.Wallet{
		WalletId:  walletId,
		AccountId: accountId,
		Type:      models.WalletTypePrimary,
		Balance:   0,
		Currency:  models.CurrencyUSD,
		Status:    models.WalletStatusInactive,
		IsDefault: true,
	})
	if err != nil {
		t.Fatalf("seed wallet %s: %v", walletId, err)
	}
}

func TestActivateAccountWallets(t *testing.T) {
	setupConsumerTestDB(t)
	seedInactiveWallet(t, "w-1", "acc-1")
	seedInactiveWallet(t, "w-2", "acc-2")

	if err := activateAccountWallets("acc-1"); err != nil {
		t.Fatalf("activateAccountWallets: %v", err)
	}

	w1, _ := db.GetWallet("w-1")
	if w1.Status != models.WalletStatusActive {
		t.Errorf("acc-1 wallet not activated, status: %s", w1.Status)
	}

	w2, _ := db.GetWallet("w-2")
	if w2.Status != models.WalletStatusInactive {
		t.Errorf("other account's wallet was touched, status: %s", w2.Status)
	}
}

func TestActivateAccountWallets_ActivatesPendingCards(t *testing.T) {
	setupConsumerTestDB(t)
	seedInactiveWallet(t, "w-3", "acc-3")

	err := db.CreateVirtualCard(&models.VirtualCard{
		ID:         "card-1",
		WalletId:   "w-3",
		CardStatus: models.VirtualCardStatusPending,
		IsActive:   false,
	})
	if err != nil {
		t.Fatalf("seed card: %v", err)
	}

	if err := activateAccountWallets("acc-3"); err != nil {
		t.Fatalf("activateAccountWallets: %v", err)
	}

	card, err := db.GetVirtualCard("card-1")
	if err != nil {
		t.Fatalf("get card: %v", err)
	}
	if card.CardStatus != models.VirtualCardStatusActive || !card.IsActive {
		t.Errorf("card not activated, status: %s isActive: %v", card.CardStatus, card.IsActive)
	}
}

func TestProcessKYCApprovedMessage(t *testing.T) {
	setupConsumerTestDB(t)
	seedInactiveWallet(t, "w-4", "acc-4")

	payload, _ := json.Marshal(map[string]string{
		"accountId": "acc-4",
		"kycstatus": "approved",
	})

	if err := processKYCApprovedMessage(kafka.Message{Value: payload}); err != nil {
		t.Fatalf("processKYCApprovedMessage: %v", err)
	}

	w, _ := db.GetWallet("w-4")
	if w.Status != models.WalletStatusActive {
		t.Errorf("wallet not activated after kyc-approved event, status: %s", w.Status)
	}
}

func TestProcessKYCApprovedMessage_IgnoresNonApproved(t *testing.T) {
	setupConsumerTestDB(t)
	seedInactiveWallet(t, "w-5", "acc-5")

	payload, _ := json.Marshal(map[string]string{
		"accountId": "acc-5",
		"kycstatus": "rejected",
	})

	if err := processKYCApprovedMessage(kafka.Message{Value: payload}); err != nil {
		t.Fatalf("processKYCApprovedMessage: %v", err)
	}

	w, _ := db.GetWallet("w-5")
	if w.Status != models.WalletStatusInactive {
		t.Errorf("wallet activated by non-approved event, status: %s", w.Status)
	}
}
