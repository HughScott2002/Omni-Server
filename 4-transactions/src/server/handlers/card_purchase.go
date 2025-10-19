package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"example.com/transactions/v1/src/db"
	"example.com/transactions/v1/src/events/producer"
	"example.com/transactions/v1/src/models"
	"example.com/transactions/v1/src/models/events"
	"example.com/transactions/v1/src/utils"
)

// HandlerCardPurchase handles simulated card purchase transactions
func HandlerCardPurchase(w http.ResponseWriter, r *http.Request) {
	var req models.PurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode purchase request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if err := utils.ValidateAmount(req.Amount); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := utils.ValidateIdempotencyKey(req.IdempotencyKey); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.CardID == "" {
		http.Error(w, "Card ID is required", http.StatusBadRequest)
		return
	}

	if req.MerchantName == "" {
		http.Error(w, "Merchant name is required", http.StatusBadRequest)
		return
	}

	// Get card information
	card, err := utils.GetVirtualCard(req.CardID)
	if err != nil {
		log.Printf("Failed to get card: %v", err)
		http.Error(w, "Card not found", http.StatusNotFound)
		return
	}

	// Get wallet information
	wallet, err := utils.GetWallet(card.WalletID)
	if err != nil {
		log.Printf("Failed to get wallet: %v", err)
		http.Error(w, "Wallet not found", http.StatusNotFound)
		return
	}

	// Check idempotency - if this request was already processed, return the same response
	cachedResponse, err := db.GetIdempotencyResponse(req.IdempotencyKey, wallet.AccountID)
	if err == nil && cachedResponse != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cachedResponse)
		return
	}

	// Check card status
	if card.CardStatus != "active" {
		response := &models.PurchaseResponse{
			Status:  "failed",
			Message: "Card is not active",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check wallet status
	if wallet.Status != "active" {
		response := &models.PurchaseResponse{
			Status:  "failed",
			Message: "Wallet is not active",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate currency
	if err := utils.ValidateCurrency(req.Currency); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check currency match (card currency must match transaction currency)
	if card.Currency != req.Currency {
		response := &models.PurchaseResponse{
			Status:  "failed",
			Message: "Currency mismatch between card and transaction",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check sufficient card balance
	if card.Balance < req.Amount {
		response := &models.PurchaseResponse{
			Status:  "failed",
			Message: "Insufficient card balance",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check card daily limit
	// TODO: Implement daily spending tracking
	if req.Amount > card.DailyLimit {
		response := &models.PurchaseResponse{
			Status:  "failed",
			Message: "Transaction exceeds daily limit",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create transaction
	now := time.Now()
	transaction := &models.Transaction{
		ID:                  utils.GenerateTransactionID(),
		Reference:           utils.GenerateTransactionReference(),
		SenderAccountID:     wallet.AccountID,
		SenderWalletID:      wallet.WalletID,
		CardID:              card.ID,
		Amount:              req.Amount,
		Currency:            req.Currency,
		TransactionType:     models.TransactionTypeCardPurchase,
		TransactionCategory: models.TransactionCategoryDebit,
		Status:              models.TransactionStatusPending,
		Description:         req.Description,
		BalanceBefore:       card.Balance,
		BalanceAfter:        card.Balance - req.Amount,
		Metadata: map[string]interface{}{
			"merchantName":     req.MerchantName,
			"merchantCategory": req.MerchantCategory,
			"idempotencyKey":   req.IdempotencyKey,
			"cardType":         card.CardType,
			"cardBrand":        card.CardBrand,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save transaction to database
	if err := db.CreateTransaction(transaction); err != nil {
		log.Printf("Failed to create transaction: %v", err)
		http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
		return
	}

	// Publish transaction created event
	producer.ProduceTransactionCreatedEvent(events.TransactionCreatedEvent{
		TransactionID:       transaction.ID,
		Reference:           transaction.Reference,
		SenderAccountID:     transaction.SenderAccountID,
		SenderWalletID:      transaction.SenderWalletID,
		Amount:              transaction.Amount,
		Currency:            transaction.Currency,
		TransactionType:     string(transaction.TransactionType),
		TransactionCategory: string(transaction.TransactionCategory),
		Status:              string(transaction.Status),
		Description:         transaction.Description,
		Timestamp:           now,
	})

	// Update balances (in production, this should be atomic)
	// TODO: Implement proper balance update in wallet service for cards
	newCardBalance := card.Balance - req.Amount
	newWalletBalance := wallet.Balance - req.Amount

	// Mark transaction as completed
	completedTime := time.Now()
	transaction.Status = models.TransactionStatusCompleted
	transaction.CompletedAt = &completedTime
	transaction.UpdatedAt = completedTime

	if err := db.UpdateTransaction(transaction); err != nil {
		log.Printf("Failed to update transaction status: %v", err)
	}

	// Publish transaction completed event
	producer.ProduceTransactionCompletedEvent(events.TransactionCompletedEvent{
		TransactionID:        transaction.ID,
		Reference:            transaction.Reference,
		SenderAccountID:      transaction.SenderAccountID,
		SenderWalletID:       transaction.SenderWalletID,
		Amount:               transaction.Amount,
		Currency:             transaction.Currency,
		TransactionType:      string(transaction.TransactionType),
		TransactionCategory:  string(transaction.TransactionCategory),
		Description:          transaction.Description,
		SenderBalanceAfter:   newCardBalance,
		Timestamp:            now,
		CompletedAt:          completedTime,
	})

	// Publish card purchase event
	producer.ProduceCardPurchaseEvent(events.CardPurchaseEvent{
		AccountID:        wallet.AccountID,
		WalletID:         wallet.WalletID,
		CardID:           card.ID,
		TransactionID:    transaction.ID,
		Reference:        transaction.Reference,
		Amount:           transaction.Amount,
		Currency:         transaction.Currency,
		MerchantName:     req.MerchantName,
		MerchantCategory: req.MerchantCategory,
		Description:      transaction.Description,
		CardBalance:      newCardBalance,
		WalletBalance:    newWalletBalance,
		Timestamp:        completedTime,
	})

	// Create response
	response := &models.PurchaseResponse{
		Status:        "success",
		Message:       "Purchase completed successfully",
		TransactionID: transaction.ID,
		Reference:     transaction.Reference,
		CardBalance:   newCardBalance,
		WalletBalance: newWalletBalance,
		Transaction:   transaction,
	}

	// Store idempotency response
	db.StoreIdempotencyKey(req.IdempotencyKey, wallet.AccountID, response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
