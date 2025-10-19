package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"omni/src/db"
	"omni/src/events/producer"
	"omni/src/models"
	"omni/src/models/events"
	"github.com/go-chi/chi/v5"
)

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

	// Update KYC status to approved
	user.KYCStatus = models.KYCStatusApproved
	err = db.UpdateUser(user)
	if err != nil {
		http.Error(w, "Error updating KYC status", http.StatusInternalServerError)
		return
	}

	// Publish KYC approved event to notify other services (e.g., wallet service)
	kycApprovedEvent := events.AccountCreatedEvent{
		AccountId: user.AccountId,
		Currency:  user.Currency,
		KYCStatus: user.KYCStatus,
	}

	err = producer.ProduceAccountCreatedEvent(kycApprovedEvent)
	if err != nil {
		log.Printf("failed to produce KYC approved event: %v", err)
	}
	log.Printf("KAFKA EVENT kyc-approved sent acc#: %s", kycApprovedEvent.AccountId)

	response := map[string]interface{}{
		"message":   "KYC approved successfully",
		"accountId": user.AccountId,
		"kycStatus": user.KYCStatus.String(),
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
