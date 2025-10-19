package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"omni/src/db"
	"omni/src/events/producer"
	"omni/src/models/events"
	"omni/src/server/middleware"
	"github.com/go-chi/chi/v5"
)

// HandlerSendContactRequest sends a contact request based on OmniTag
func HandlerSendContactRequest(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		OmniTag string `json:"omniTag"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if requestBody.OmniTag == "" {
		http.Error(w, "OmniTag is required", http.StatusBadRequest)
		return
	}

	// Get current user's account ID from auth context
	requesterAccountID, ok := middleware.GetAccountIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Find addressee by OmniTag
	addressee, err := db.GetUserByOmniTag(requestBody.OmniTag)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if trying to add yourself
	if addressee.AccountId == requesterAccountID {
		http.Error(w, "Cannot send contact request to yourself", http.StatusBadRequest)
		return
	}

	// Send contact request
	contact, err := db.SendContactRequest(requesterAccountID, addressee.AccountId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	// Publish contact request sent event
	event := events.ContactRequestSentEvent{
		ContactID:   contact.ID,
		RequesterID: requesterAccountID,
		AddresseeID: addressee.AccountId,
		OmniTag:     addressee.OmniTag,
		Timestamp:   time.Now(),
	}
	if err := producer.ProduceContactRequestSentEvent(event); err != nil {
		log.Printf("Failed to publish contact request sent event: %v", err)
		// Don't fail the request if event publishing fails
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Contact request sent successfully",
		"contactId": contact.ID,
		"omniTag":   addressee.OmniTag,
	})
}

// HandlerGetContacts returns all accepted contacts for the user
func HandlerGetContacts(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "accountid")
	if accountID == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	// Verify that the requesting user is authorized to view these contacts
	currentUserAccountID, ok := middleware.GetAccountIDFromContext(r)
	if !ok || currentUserAccountID != accountID {
		http.Error(w, "Unauthorized: You can only view your own contacts", http.StatusForbidden)
		return
	}

	contacts, err := db.GetContactsByUser(accountID)
	if err != nil {
		http.Error(w, "Failed to retrieve contacts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"contacts": contacts,
		"count":    len(contacts),
	})
}

// HandlerGetPendingRequests returns pending contact requests for the user
func HandlerGetPendingRequests(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "accountid")
	if accountID == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	// Verify that the requesting user is authorized
	currentUserAccountID, ok := middleware.GetAccountIDFromContext(r)
	if !ok || currentUserAccountID != accountID {
		http.Error(w, "Unauthorized: You can only view your own pending requests", http.StatusForbidden)
		return
	}

	requests, err := db.GetPendingRequests(accountID)
	if err != nil {
		http.Error(w, "Failed to retrieve pending requests", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"requests": requests,
		"count":    len(requests),
	})
}

// HandlerGetSentRequests returns sent contact requests for the user
func HandlerGetSentRequests(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "accountid")
	if accountID == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	// Verify that the requesting user is authorized
	currentUserAccountID, ok := middleware.GetAccountIDFromContext(r)
	if !ok || currentUserAccountID != accountID {
		http.Error(w, "Unauthorized: You can only view your own sent requests", http.StatusForbidden)
		return
	}

	requests, err := db.GetSentRequests(accountID)
	if err != nil {
		http.Error(w, "Failed to retrieve sent requests", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"requests": requests,
		"count":    len(requests),
	})
}

// HandlerAcceptContactRequest accepts a contact request
func HandlerAcceptContactRequest(w http.ResponseWriter, r *http.Request) {
	contactID := chi.URLParam(r, "contactid")
	if contactID == "" {
		http.Error(w, "Contact ID is required", http.StatusBadRequest)
		return
	}

	// Get current user's account ID from auth context
	userAccountID, ok := middleware.GetAccountIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := db.AcceptContactRequest(contactID, userAccountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get contact details for the event
	contact, err := db.GetContact(contactID)
	if err != nil {
		log.Printf("Failed to get contact for event: %v", err)
	} else {
		// Publish contact request accepted event
		event := events.ContactRequestAcceptedEvent{
			ContactID:   contactID,
			RequesterID: contact.RequesterID,
			AddresseeID: contact.AddresseeID,
			AcceptedBy:  userAccountID,
			Timestamp:   time.Now(),
		}
		if err := producer.ProduceContactRequestAcceptedEvent(event); err != nil {
			log.Printf("Failed to publish contact request accepted event: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Contact request accepted successfully",
	})
}

// HandlerRejectContactRequest rejects a contact request
func HandlerRejectContactRequest(w http.ResponseWriter, r *http.Request) {
	contactID := chi.URLParam(r, "contactid")
	if contactID == "" {
		http.Error(w, "Contact ID is required", http.StatusBadRequest)
		return
	}

	// Get current user's account ID from auth context
	userAccountID, ok := middleware.GetAccountIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := db.RejectContactRequest(contactID, userAccountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get contact details for the event
	contact, err := db.GetContact(contactID)
	if err != nil {
		log.Printf("Failed to get contact for event: %v", err)
	} else {
		// Publish contact request rejected event
		event := events.ContactRequestRejectedEvent{
			ContactID:   contactID,
			RequesterID: contact.RequesterID,
			AddresseeID: contact.AddresseeID,
			RejectedBy:  userAccountID,
			Timestamp:   time.Now(),
		}
		if err := producer.ProduceContactRequestRejectedEvent(event); err != nil {
			log.Printf("Failed to publish contact request rejected event: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Contact request rejected successfully",
	})
}

// HandlerBlockContact blocks a contact
func HandlerBlockContact(w http.ResponseWriter, r *http.Request) {
	contactID := chi.URLParam(r, "contactid")
	if contactID == "" {
		http.Error(w, "Contact ID is required", http.StatusBadRequest)
		return
	}

	// Get current user's account ID from auth context
	userAccountID, ok := middleware.GetAccountIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := db.BlockContact(contactID, userAccountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get contact details for the event
	contact, err := db.GetContact(contactID)
	if err != nil {
		log.Printf("Failed to get contact for event: %v", err)
	} else {
		// Publish contact blocked event
		event := events.ContactBlockedEvent{
			ContactID:   contactID,
			RequesterID: contact.RequesterID,
			AddresseeID: contact.AddresseeID,
			BlockedBy:   userAccountID,
			Timestamp:   time.Now(),
		}
		if err := producer.ProduceContactBlockedEvent(event); err != nil {
			log.Printf("Failed to publish contact blocked event: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Contact blocked successfully",
	})
}

// HandlerDeleteContact deletes a contact
func HandlerDeleteContact(w http.ResponseWriter, r *http.Request) {
	contactID := chi.URLParam(r, "contactid")
	if contactID == "" {
		http.Error(w, "Contact ID is required", http.StatusBadRequest)
		return
	}

	// Get current user's account ID from auth context
	userAccountID, ok := middleware.GetAccountIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := db.DeleteContact(contactID, userAccountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Contact deleted successfully",
	})
}
