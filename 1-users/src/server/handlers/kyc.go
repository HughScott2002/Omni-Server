package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"omni/src/db"
	"omni/src/events/producer"
	"omni/src/models"
	"omni/src/models/events"
)

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

	// The user must explicitly confirm their information is accurate and
	// authorize its processing before verification can proceed.
	consent, _ := kycData["consent"].(bool)
	if !consent {
		http.Error(w, "Consent is required to submit KYC", http.StatusBadRequest)
		return
	}

	// Verification currently needs the identity fields collected at
	// registration; document upload and manual review come later.
	missing := []string{}
	required := map[string]string{
		"firstName":  user.FirstName,
		"lastName":   user.LastName,
		"address":    user.Address,
		"city":       user.City,
		"country":    user.Country,
		"postalCode": user.PostalCode,
		"dob":        user.DOB,
		"govId":      user.GovId,
	}
	for field, value := range required {
		if value == "" {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":       "Missing required KYC information",
			"missingFields": missing,
		})
		return
	}

	user.DataAuthorization = true
	if err := db.UpdateUser(user); err != nil {
		http.Error(w, "Error updating KYC information", http.StatusInternalServerError)
		return
	}

	// With complete registration info + signed consent, approve immediately.
	// A real verification pipeline (documents, review) will replace this and
	// leave the status pending until review completes.
	if err := approveKYCByAccountId(user.AccountId); err != nil {
		http.Error(w, "Error approving KYC", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":   "Identity verified. Your account is approved.",
		"accountId": user.AccountId,
		"kycStatus": models.KYCStatusApproved.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
