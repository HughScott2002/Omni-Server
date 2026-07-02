package handlers

import (
	"testing"

	"omni/src/db"
	"omni/src/models"
)

func TestApproveKYCByAccountId_ApprovesPendingUser(t *testing.T) {
	setupTestDB()

	user, _ := createTestUser("kyc-approve@example.com", "password123")

	if err := approveKYCByAccountId(user.AccountId); err != nil {
		t.Fatalf("approveKYCByAccountId: %v", err)
	}

	updated, _ := db.GetUser(user.Email)
	if updated.KYCStatus != models.KYCStatusApproved {
		t.Errorf("expected KYC approved, got %s", updated.KYCStatus.String())
	}
}

func TestApproveKYCByAccountId_IdempotentWhenApproved(t *testing.T) {
	setupTestDB()

	user, _ := createTestUser("kyc-idempotent@example.com", "password123")
	user.KYCStatus = models.KYCStatusApproved
	db.UpdateUser(user)

	if err := approveKYCByAccountId(user.AccountId); err != nil {
		t.Fatalf("expected idempotent approval, got error: %v", err)
	}
}

func TestApproveKYCByAccountId_UnknownAccount(t *testing.T) {
	setupTestDB()

	if err := approveKYCByAccountId("NO-SUCH-ACCOUNT"); err == nil {
		t.Error("expected error for unknown account")
	}
}
