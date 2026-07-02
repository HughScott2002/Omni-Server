package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"omni/src/db"
	"omni/src/events/producer"
	"omni/src/models"
	"omni/src/models/events"
	"github.com/go-chi/chi/v5"
)

// TEMPORARY: KYC auto-approves this long after registration so wallets
// activate without manual review. Replace with an admin-driven KYC service
// producing the same kyc-approved event.
const kycAutoApprovalDelay = time.Minute

// approveKYCByAccountId marks the user approved and notifies other services
// (the wallet service activates the account's wallets). Idempotent: an
// already-approved account is a no-op.
func approveKYCByAccountId(accountId string) error {
	user, err := db.GetUserByAccountId(accountId)
	if err != nil {
		return err
	}

	if user.KYCStatus == models.KYCStatusApproved {
		return nil
	}

	user.KYCStatus = models.KYCStatusApproved
	if err := db.UpdateUser(user); err != nil {
		return err
	}

	kycApprovedEvent := events.KYCApprovedEvent{
		AccountId: user.AccountId,
		KYCStatus: user.KYCStatus,
	}
	if err := producer.ProduceKYCApprovedEvent(kycApprovedEvent); err != nil {
		log.Printf("failed to produce kyc-approved event: %v", err)
	} else {
		log.Printf("KAFKA EVENT kyc-approved sent acc#: %s", user.AccountId)
	}

	return nil
}

// ScheduleAutoKYCApproval auto-approves a freshly registered account after
// kycAutoApprovalDelay unless its status changed in the meantime (e.g. an
// admin rejected it).
func ScheduleAutoKYCApproval(accountId string) {
	go func() {
		time.Sleep(kycAutoApprovalDelay)

		user, err := db.GetUserByAccountId(accountId)
		if err != nil {
			log.Printf("auto KYC approval: account %s not found: %v", accountId, err)
			return
		}
		if user.KYCStatus != models.KYCStatusPending {
			log.Printf("auto KYC approval: skipping acc#%s, status is %s", accountId, user.KYCStatus.String())
			return
		}

		if err := approveKYCByAccountId(accountId); err != nil {
			log.Printf("auto KYC approval failed for acc#%s: %v", accountId, err)
			return
		}
		log.Printf("auto KYC approval: approved acc#%s", accountId)
	}()
}

// HandlerApproveKYC approves a user's KYC status
// This should typically be called after manual review or automated verification
func HandlerApproveKYC(w http.ResponseWriter, r *http.Request) {
	accountId := chi.URLParam(r, "accountid")

	if accountId == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	// Get the user
	user, err := db.GetUserByAccountId(accountId)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if KYC is already approved
	if user.KYCStatus == models.KYCStatusApproved {
		http.Error(w, "KYC already approved", http.StatusConflict)
		return
	}

	if err := approveKYCByAccountId(user.AccountId); err != nil {
		http.Error(w, "Error updating KYC status", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":   "KYC approved successfully",
		"accountId": user.AccountId,
		"kycStatus": models.KYCStatusApproved.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandlerRejectKYC rejects a user's KYC status
func HandlerRejectKYC(w http.ResponseWriter, r *http.Request) {
	accountId := chi.URLParam(r, "accountid")

	if accountId == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	// Get the user
	user, err := db.GetUserByAccountId(accountId)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Update KYC status to rejected
	user.KYCStatus = models.KYCStatusRejected
	err = db.UpdateUser(user)
	if err != nil {
		http.Error(w, "Error updating KYC status", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":   "KYC rejected",
		"accountId": user.AccountId,
		"kycStatus": user.KYCStatus.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandlerGetKYCStatus gets the current KYC status for a user
func HandlerGetKYCStatus(w http.ResponseWriter, r *http.Request) {
	accountId := chi.URLParam(r, "accountid")

	if accountId == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	// Get the user
	user, err := db.GetUserByAccountId(accountId)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"accountId": user.AccountId,
		"kycStatus": user.KYCStatus.String(),
		"omniTag":   user.OmniTag,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandlerSubmitKYC allows a user to submit or update their KYC information
func HandlerSubmitKYC(w http.ResponseWriter, r *http.Request) {
	accountId := chi.URLParam(r, "accountid")

	if accountId == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var kycData map[string]interface{}
	if err := json.Unmarshal(body, &kycData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the user
	user, err := db.GetUserByAccountId(accountId)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Update KYC fields
	if firstName, ok := kycData["firstName"].(string); ok {
		user.FirstName = firstName
	}
	if lastName, ok := kycData["lastName"].(string); ok {
		user.LastName = lastName
	}
	if phone, ok := kycData["phone"].(string); ok {
		user.Phone = phone
	}
	if address, ok := kycData["address"].(string); ok {
		user.Address = address
	}
	if city, ok := kycData["city"].(string); ok {
		user.City = city
	}
	if state, ok := kycData["state"].(string); ok {
		user.State = state
	}
	if country, ok := kycData["country"].(string); ok {
		user.Country = country
	}
	if postalCode, ok := kycData["postalCode"].(string); ok {
		user.PostalCode = postalCode
	}
	if dob, ok := kycData["dob"].(string); ok {
		user.DOB = dob
	}
	if govId, ok := kycData["govId"].(string); ok {
		user.GovId = govId
	}

	// Set KYC status to pending (requires approval)
	user.KYCStatus = models.KYCStatusPending

	err = db.UpdateUser(user)
	if err != nil {
		http.Error(w, "Error updating KYC information", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":   "KYC information submitted successfully. Awaiting approval.",
		"accountId": user.AccountId,
		"kycStatus": user.KYCStatus.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
