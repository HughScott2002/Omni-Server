package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"omni/src/db"
	"omni/src/events/producer"
	"omni/src/models"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

func HandlerGetUserProfile(w http.ResponseWriter, r *http.Request) {
	accountId := chi.URLParam(r, "accountid")
	if accountId == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	users, err := db.GetUserByAccountId(accountId) // You'll need to implement this
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	//Type of account
	//TODO: Omni Tag, Autogenerate it for now

	profile := map[string]interface{}{
		"accountId":  users.AccountId,
		"email":      users.Email,
		"firstName":  users.FirstName,
		"lastName":   users.LastName,
		"phone":      users.Phone,
		"address":    users.Address,
		"city":       users.City,
		"state":      users.State,
		"country":    users.Country,
		"currency":   users.Currency,
		"postalCode": users.PostalCode,
		"dob":        users.DOB,
		"govId":      users.GovId,
		"kycStatus":  users.KYCStatus.String(),
		// "backupCodes": users.BackupCodes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func HandlerUpdateUserProfile(w http.ResponseWriter, r *http.Request) {
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

	var updateRequest models.User
	if err := json.Unmarshal(body, &updateRequest); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	currentUser, err := db.GetUserByAccountId(accountId)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Update only allowed fields
	currentUser.FirstName = updateRequest.FirstName
	currentUser.LastName = updateRequest.LastName
	currentUser.Phone = updateRequest.Phone
	currentUser.Address = updateRequest.Address
	currentUser.City = updateRequest.City
	currentUser.State = updateRequest.State
	currentUser.Country = updateRequest.Country
	currentUser.PostalCode = updateRequest.PostalCode
	currentUser.Currency = updateRequest.Currency
	currentUser.DOB = updateRequest.DOB
	currentUser.GovId = updateRequest.GovId

	if err := db.UpdateUser(currentUser); err != nil {
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "Profile updated successfully",
		"user": map[string]interface{}{
			"firstName":  currentUser.FirstName,
			"lastName":   currentUser.LastName,
			"phone":      currentUser.Phone,
			"address":    currentUser.Address,
			"city":       currentUser.City,
			"state":      currentUser.State,
			"country":    currentUser.Country,
			"currency":   currentUser.Currency,
			"postalCode": currentUser.PostalCode,
			"dob":        currentUser.DOB,
			"govId":      currentUser.GovId,
			"email":      currentUser.Email,
			"kycStatus":  currentUser.KYCStatus.String(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func HandlerDeleteUserAccount(w http.ResponseWriter, r *http.Request) {
	accountId := chi.URLParam(r, "accountid")
	if accountId == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	user, err := db.GetUserByAccountId(accountId)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if user.Status != models.AccountStatusActive {
		http.Error(w, "Account is already disabled or pending deletion", http.StatusBadRequest)
		return
	}

	// Set account status and deletion time
	now := time.Now()
	scheduledDeletion := now.AddDate(0, 0, 30) // 30 days from now
	user.Status = models.AccountStatusPendingDeletion
	user.DeletionRequestedAt = &now

	// Update user in database
	if err := db.UpdateUser(user); err != nil {
		http.Error(w, "Failed to update user status", http.StatusInternalServerError)
		return
	}

	// Publish Kafka event
	event := producer.AccountDeletionRequestedEvent{
		AccountId:         user.AccountId,
		Email:             user.Email,
		RequestedAt:       now,
		ScheduledDeletion: scheduledDeletion,
	}

	if err := producer.ProduceAccountDeletionRequestedEvent(event); err != nil {
		log.Printf("Failed to produce account deletion event: %v", err)
		// Continue execution - event failure shouldn't block account disable
	}

	// Invalidate all sessions
	if err := db.DeleteUserSessions(user.Email); err != nil {
		log.Printf("Failed to delete user sessions: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":           "Account scheduled for deletion",
		"scheduledDeletion": scheduledDeletion,
	})
}

func HandlerChangePassword(w http.ResponseWriter, r *http.Request) {
	// Get device information
	deviceInfo := r.Header.Get("User-Agent")

	fmt.Printf("%s", deviceInfo)

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse the JSON body
	var passwordChangeReq models.PasswordChangeRequest
	if err := json.Unmarshal(body, &passwordChangeReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate passwords match
	if passwordChangeReq.NewPassword != passwordChangeReq.ConfirmNewPassword {
		http.Error(w, "New passwords do not match", http.StatusBadRequest)
		return
	}

	// Get the user from database
	storedUser, err := db.GetUser(passwordChangeReq.Email)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify account ID matches
	if storedUser.AccountId != passwordChangeReq.AccountId {
		http.Error(w, "Invalid account ID", http.StatusUnauthorized)
		return
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(storedUser.HashedPassword), []byte(passwordChangeReq.CurrentPassword))
	if err != nil {
		http.Error(w, "Current password is incorrect", http.StatusUnauthorized)
		return
	}

	// Check if new password is same as current password
	err = bcrypt.CompareHashAndPassword([]byte(storedUser.HashedPassword), []byte(passwordChangeReq.NewPassword))
	if err == nil {
		http.Error(w, "New password cannot be the same as current password", http.StatusBadRequest)
		return
	}

	// Hash the new password
	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(passwordChangeReq.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing new password", http.StatusInternalServerError)
		return
	}

	// Update user's password
	storedUser.HashedPassword = string(hashedNewPassword)
	err = db.UpdateUser(storedUser)
	if err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	// Invalidate all refresh tokens for this user
	err = db.DeleteUserSessions(passwordChangeReq.Email)
	if err != nil {
		log.Printf("Error deleting user sessions: %v", err)
	}

	// TODO: Produce password changed event
	// passwordChangedEvent := events.PasswordChangedEvent{
	// 	AccountId:  storedUser.AccountId,
	// 	DeviceInfo: deviceInfo,
	// 	ChangedAt:  time.Now(),
	// }
	// err = producer.ProducePasswordChangedEvent(passwordChangedEvent)
	// if err != nil {
	// 	log.Printf("Failed to produce password changed event: %v", err)
	// }

	// Return success response
	response := map[string]string{
		"message":   "Password changed successfully",
		"email":     passwordChangeReq.Email,
		"accountId": passwordChangeReq.AccountId,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
