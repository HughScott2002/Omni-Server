package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"example.com/m/v2/src/db"
	"example.com/m/v2/src/events/producer"
	"example.com/m/v2/src/models"
	"example.com/m/v2/src/models/events"
	"example.com/m/v2/src/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// HandlerCreateVirtualCard creates a new virtual card
func HandlerCreateVirtualCard(w http.ResponseWriter, r *http.Request) {
	var req models.VirtualCardCreate

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate wallet exists
	wallet, err := db.GetWallet(req.WalletId)
	if err != nil {
		http.Error(w, "Wallet not found", http.StatusNotFound)
		return
	}

	// Generate card number and CVV
	cardNumber, err := utils.GenerateVisaCardNumber()
	if err != nil {
		http.Error(w, "Failed to generate card number", http.StatusInternalServerError)
		return
	}

	cvv, cvvHash, err := utils.GenerateCVV()
	if err != nil {
		http.Error(w, "Failed to generate CVV", http.StatusInternalServerError)
		return
	}

	// Generate expiry date (3 years from now)
	expiryDate := utils.GenerateCardExpiryDate()

	// Create virtual card
	card := &models.VirtualCard{
		ID:               uuid.New().String(),
		WalletId:         req.WalletId,
		CardType:         req.CardType,
		CardBrand:        req.CardBrand,
		Currency:         req.Currency,
		CardStatus:       models.VirtualCardStatusPending,
		DailyLimit:       req.DailyLimit,
		MonthlyLimit:     req.MonthlyLimit,
		NameOnCard:       req.NameOnCard,
		CardNumber:       cardNumber,
		CVVHash:          cvvHash,
		ExpiryDate:       expiryDate,
		IsActive:         true,
		AvailableBalance: 0,
		TotalToppedUp:    0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := db.CreateVirtualCard(card); err != nil {
		log.Printf("Failed to create virtual card: %v", err)
		http.Error(w, "Failed to create virtual card", http.StatusInternalServerError)
		return
	}

	// Publish virtual card created event
	event := events.VirtualCardCreatedEvent{
		CardID:         card.ID,
		AccountID:      wallet.AccountId,
		WalletId:       card.WalletId,
		CardType:       string(card.CardType),
		Currency:       string(card.Currency),
		LastFourDigits: card.LastFourDigits(),
		Timestamp:      time.Now(),
	}
	if err := producer.ProduceVirtualCardCreatedEvent(event); err != nil {
		log.Printf("Failed to publish virtual card created event: %v", err)
	}

	// Return response with CVV (only time CVV is returned)
	response := map[string]interface{}{
		"message":          "Virtual card created successfully",
		"card":             card,
		"cvv":              cvv, // CVV only returned once
		"maskedCardNumber": card.MaskedCardNumber(),
		"lastFourDigits":   card.LastFourDigits(),
	}

	// Add to account's card list
	if wallet.AccountId != "" {
		log.Printf("Virtual card %s created for account %s", card.ID, wallet.AccountId)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// HandlerGetVirtualCard gets a virtual card by ID
func HandlerGetVirtualCard(w http.ResponseWriter, r *http.Request) {
	// Try both parameter extraction methods
	cardID := chi.URLParam(r, "cardid")
	if cardID == "" {
		// Try v5 method
		ctx := chi.RouteContext(r.Context())
		if ctx != nil {
			cardID = ctx.URLParam("cardid")
		}
	}
	log.Printf("GetVirtualCard - URL: %s, cardID from param: '%s', all params: %v", r.URL.Path, cardID, chi.RouteContext(r.Context()))
	if cardID == "" {
		http.Error(w, "Card ID is required", http.StatusBadRequest)
		return
	}

	card, err := db.GetVirtualCard(cardID)
	if err != nil {
		http.Error(w, "Virtual card not found", http.StatusNotFound)
		return
	}

	// Remove sensitive data
	card.CardNumber = card.MaskedCardNumber()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

// HandlerGetVirtualCardsByAccount gets all virtual cards for an account
func HandlerGetVirtualCardsByAccount(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "accountid")
	if accountID == "" {
		// Try v5 method
		ctx := chi.RouteContext(r.Context())
		if ctx != nil {
			accountID = ctx.URLParam("accountid")
		}
	}
	log.Printf("GetVirtualCardsByAccount - URL: %s, accountID from param: '%s', RouteContext: %v", r.URL.Path, accountID, chi.RouteContext(r.Context()))
	if accountID == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	cards, err := db.GetVirtualCardsByAccountId(accountID)
	if err != nil {
		http.Error(w, "Failed to retrieve virtual cards", http.StatusInternalServerError)
		return
	}

	// Mask card numbers
	for _, card := range cards {
		card.CardNumber = card.MaskedCardNumber()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cards": cards,
		"count": len(cards),
	})
}

// HandlerUpdateVirtualCard updates a virtual card
func HandlerUpdateVirtualCard(w http.ResponseWriter, r *http.Request) {
	cardID := chi.URLParam(r, "cardid")
	if cardID == "" {
		http.Error(w, "Card ID is required", http.StatusBadRequest)
		return
	}

	var req models.VirtualCardUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	card, err := db.GetVirtualCard(cardID)
	if err != nil {
		http.Error(w, "Virtual card not found", http.StatusNotFound)
		return
	}

	// Update fields if provided
	if req.DailyLimit != nil {
		card.DailyLimit = *req.DailyLimit
	}
	if req.MonthlyLimit != nil {
		card.MonthlyLimit = *req.MonthlyLimit
	}
	if req.IsActive != nil {
		card.IsActive = *req.IsActive
		if !*req.IsActive {
			card.CardStatus = models.VirtualCardStatusInactive
		} else {
			card.CardStatus = models.VirtualCardStatusActive
		}
	}

	if err := db.UpdateVirtualCard(card); err != nil {
		http.Error(w, "Failed to update virtual card", http.StatusInternalServerError)
		return
	}

	card.CardNumber = card.MaskedCardNumber()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Virtual card updated successfully",
		"card":    card,
	})
}

// HandlerBlockVirtualCard blocks a virtual card
func HandlerBlockVirtualCard(w http.ResponseWriter, r *http.Request) {
	cardID := chi.URLParam(r, "cardid")
	if cardID == "" {
		http.Error(w, "Card ID is required", http.StatusBadRequest)
		return
	}

	var req models.CardBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Get blockedBy from auth context
	blockedBy := "system" // Replace with actual user ID from auth

	// Get card to get account ID for event
	card, err := db.GetVirtualCard(cardID)
	if err != nil {
		http.Error(w, "Virtual card not found", http.StatusNotFound)
		return
	}

	wallet, err := db.GetWallet(card.WalletId)
	if err != nil {
		http.Error(w, "Wallet not found", http.StatusNotFound)
		return
	}

	if err := db.BlockVirtualCard(cardID, req.BlockReason, req.BlockReasonDesc, blockedBy); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Publish virtual card blocked event
	event := events.VirtualCardBlockedEvent{
		CardID:      cardID,
		AccountID:   wallet.AccountId,
		BlockReason: string(req.BlockReason),
		BlockedBy:   blockedBy,
		Timestamp:   time.Now(),
	}
	if err := producer.ProduceVirtualCardBlockedEvent(event); err != nil {
		log.Printf("Failed to publish virtual card blocked event: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Virtual card blocked successfully",
	})
}

// HandlerTopUpVirtualCard tops up a virtual card
func HandlerTopUpVirtualCard(w http.ResponseWriter, r *http.Request) {
	cardID := chi.URLParam(r, "cardid")
	if cardID == "" {
		http.Error(w, "Card ID is required", http.StatusBadRequest)
		return
	}

	var req models.CardTopUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		http.Error(w, "Amount must be greater than zero", http.StatusBadRequest)
		return
	}

	if err := db.TopUpVirtualCard(cardID, req.Amount); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	card, _ := db.GetVirtualCard(cardID)
	wallet, _ := db.GetWallet(card.WalletId)

	// Publish virtual card topped up event
	if wallet != nil {
		event := events.VirtualCardToppedUpEvent{
			CardID:     cardID,
			AccountID:  wallet.AccountId,
			Amount:     req.Amount,
			NewBalance: card.AvailableBalance,
			Timestamp:  time.Now(),
		}
		if err := producer.ProduceVirtualCardToppedUpEvent(event); err != nil {
			log.Printf("Failed to publish virtual card topped up event: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.CardTopUpResponse{
		Status:  "success",
		Message: "Card topped up successfully",
		Data: map[string]interface{}{
			"newBalance": card.AvailableBalance,
			"amount":     req.Amount,
		},
	})
}

// HandlerRequestPhysicalCard requests a physical card
func HandlerRequestPhysicalCard(w http.ResponseWriter, r *http.Request) {
	cardID := chi.URLParam(r, "cardid")
	if cardID == "" {
		http.Error(w, "Card ID is required", http.StatusBadRequest)
		return
	}

	var req models.PhysicalCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get card and wallet for event
	card, err := db.GetVirtualCard(cardID)
	if err != nil {
		http.Error(w, "Virtual card not found", http.StatusNotFound)
		return
	}

	wallet, err := db.GetWallet(card.WalletId)
	if err != nil {
		http.Error(w, "Wallet not found", http.StatusNotFound)
		return
	}

	if err := db.RequestPhysicalCard(cardID, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Publish physical card requested event
	event := events.PhysicalCardRequestedEvent{
		CardID:          cardID,
		AccountID:       wallet.AccountId,
		DeliveryAddress: req.DeliveryAddress,
		DeliveryCity:    req.DeliveryCity,
		DeliveryCountry: req.DeliveryCountry,
		Timestamp:       time.Now(),
	}
	if err := producer.ProducePhysicalCardRequestedEvent(event); err != nil {
		log.Printf("Failed to publish physical card requested event: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Physical card request submitted successfully",
		"status":  "pending",
	})
}

// HandlerDeleteVirtualCard deletes a virtual card
func HandlerDeleteVirtualCard(w http.ResponseWriter, r *http.Request) {
	cardID := chi.URLParam(r, "cardid")
	if cardID == "" {
		http.Error(w, "Card ID is required", http.StatusBadRequest)
		return
	}

	// Get card and wallet for event before deletion
	card, err := db.GetVirtualCard(cardID)
	if err != nil {
		http.Error(w, "Virtual card not found", http.StatusNotFound)
		return
	}

	wallet, err := db.GetWallet(card.WalletId)
	if err != nil {
		http.Error(w, "Wallet not found", http.StatusNotFound)
		return
	}

	lastFourDigits := card.LastFourDigits()

	if err := db.DeleteVirtualCard(cardID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Publish virtual card deleted event
	event := events.VirtualCardDeletedEvent{
		CardID:         cardID,
		AccountID:      wallet.AccountId,
		LastFourDigits: lastFourDigits,
		Timestamp:      time.Now(),
	}
	if err := producer.ProduceVirtualCardDeletedEvent(event); err != nil {
		log.Printf("Failed to publish virtual card deleted event: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Virtual card deleted successfully",
		"deletedAt": time.Now(),
	})
}
