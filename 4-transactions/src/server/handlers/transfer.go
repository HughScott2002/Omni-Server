package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"example.com/transactions/v1/src/db"
	"example.com/transactions/v1/src/events/producer"
	"example.com/transactions/v1/src/models"
	"example.com/transactions/v1/src/models/events"
	"example.com/transactions/v1/src/utils"
)

// HandlerTransferMoney handles wallet-to-wallet money transfers using OmniTag
func HandlerTransferMoney(w http.ResponseWriter, r *http.Request) {
	var req models.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode transfer request: %v", err)
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

	if req.SenderWalletID == "" {
		http.Error(w, "Sender wallet ID is required", http.StatusBadRequest)
		return
	}

	if req.ReceiverOmniTag == "" {
		http.Error(w, "Receiver OmniTag is required", http.StatusBadRequest)
		return
	}

	// Get sender wallet
	senderWallet, err := utils.GetWallet(req.SenderWalletID)
	if err != nil {
		log.Printf("Failed to get sender wallet: %v", err)
		http.Error(w, "Sender wallet not found", http.StatusNotFound)
		return
	}

	// Check idempotency - if this request was already processed, return the same response
	cachedResponse, err := db.GetIdempotencyResponse(req.IdempotencyKey, senderWallet.AccountID)
	if err == nil && cachedResponse != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cachedResponse)
		return
	}

	// Check wallet status
	if senderWallet.Status != "active" {
		response := &models.TransferResponse{
			Status:  "failed",
			Message: "Sender wallet is not active",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check sufficient balance
	if senderWallet.Balance < req.Amount {
		response := &models.TransferResponse{
			Status:  "failed",
			Message: "Insufficient balance",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Look up receiver by OmniTag
	receiverUser, err := utils.GetUserByOmniTag(req.ReceiverOmniTag)
	if err != nil {
		log.Printf("Failed to find receiver by OmniTag %s: %v", req.ReceiverOmniTag, err)
		response := &models.TransferResponse{
			Status:  "failed",
			Message: "Receiver not found",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Prevent self-transfer
	if senderWallet.AccountID == receiverUser.AccountID {
		response := &models.TransferResponse{
			Status:  "failed",
			Message: "Cannot transfer to yourself",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get receiver's default wallet
	receiverWallet, err := utils.GetDefaultWallet(receiverUser.AccountID)
	if err != nil {
		log.Printf("Failed to get receiver wallet for account %s: %v", receiverUser.AccountID, err)
		response := &models.TransferResponse{
			Status:  "failed",
			Message: "Receiver wallet not found",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check currency match
	if senderWallet.Currency != receiverWallet.Currency {
		response := &models.TransferResponse{
			Status:  "failed",
			Message: fmt.Sprintf("Currency mismatch: sender has %s, receiver has %s", senderWallet.Currency, receiverWallet.Currency),
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
		SenderAccountID:     senderWallet.AccountID,
		ReceiverAccountID:   receiverUser.AccountID,
		SenderWalletID:      senderWallet.WalletID,
		ReceiverWalletID:    receiverWallet.WalletID,
		Amount:              req.Amount,
		Currency:            senderWallet.Currency,
		TransactionType:     models.TransactionTypeTransfer,
		TransactionCategory: models.TransactionCategoryDebit, // From sender's perspective
		Status:              models.TransactionStatusPending,
		Description:         req.Description,
		BalanceBefore:       senderWallet.Balance,
		BalanceAfter:        senderWallet.Balance - req.Amount,
		Metadata: map[string]interface{}{
			"receiverOmniTag": req.ReceiverOmniTag,
			"idempotencyKey":  req.IdempotencyKey,
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
		ReceiverAccountID:   transaction.ReceiverAccountID,
		SenderWalletID:      transaction.SenderWalletID,
		ReceiverWalletID:    transaction.ReceiverWalletID,
		Amount:              transaction.Amount,
		Currency:            transaction.Currency,
		TransactionType:     string(transaction.TransactionType),
		TransactionCategory: string(transaction.TransactionCategory),
		Status:              string(transaction.Status),
		Description:         transaction.Description,
		Timestamp:           now,
	})

	// FRAUD DETECTION: Assess transaction risk
	riskAssessment, err := utils.AssessTransactionRisk(utils.RiskAssessmentRequest{
		TransactionID:     transaction.ID,
		SenderAccountID:   transaction.SenderAccountID,
		ReceiverAccountID: transaction.ReceiverAccountID,
		Amount:            transaction.Amount,
		Currency:          transaction.Currency,
		TransactionType:   string(transaction.TransactionType),
		Description:       transaction.Description,
		Metadata:          transaction.Metadata,
	})

	if err != nil {
		log.Printf("Failed to assess transaction risk: %v", err)
		// In production, you might want to fail the transaction or flag for manual review
		// For now, we'll log the error and continue
	} else if !utils.IsTransactionApproved(riskAssessment) {
		// Transaction was declined by fraud detection
		log.Printf("Transaction %s declined by fraud detection: decision=%s, riskLevel=%s, reasons=%v",
			transaction.ID, riskAssessment.Decision, riskAssessment.RiskLevel, riskAssessment.Reasons)

		// Update transaction status to failed
		failedTime := time.Now()
		transaction.Status = models.TransactionStatusFailed
		transaction.UpdatedAt = failedTime

		// Add risk assessment info to metadata
		transaction.Metadata["riskScore"] = riskAssessment.RiskScore
		transaction.Metadata["riskLevel"] = riskAssessment.RiskLevel
		transaction.Metadata["riskDecision"] = riskAssessment.Decision
		transaction.Metadata["riskReasons"] = riskAssessment.Reasons

		if err := db.UpdateTransaction(transaction); err != nil {
			log.Printf("Failed to update transaction status: %v", err)
		}

		// Publish transaction failed event
		producer.ProduceTransactionFailedEvent(events.TransactionFailedEvent{
			TransactionID:       transaction.ID,
			Reference:           transaction.Reference,
			SenderAccountID:     transaction.SenderAccountID,
			ReceiverAccountID:   transaction.ReceiverAccountID,
			Amount:              transaction.Amount,
			Currency:            transaction.Currency,
			TransactionType:     string(transaction.TransactionType),
			TransactionCategory: string(transaction.TransactionCategory),
			Description:         transaction.Description,
			FailedReason:        fmt.Sprintf("Declined by fraud detection: %s", riskAssessment.RiskLevel),
			Timestamp:           failedTime,
		})

		response := &models.TransferResponse{
			Status:  "failed",
			Message: fmt.Sprintf("Transaction declined due to risk assessment: %s", riskAssessment.RiskLevel),
		}

		// Store idempotency response
		db.StoreIdempotencyKey(req.IdempotencyKey, senderWallet.AccountID, response)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Transaction approved - add risk assessment info to metadata
	if riskAssessment != nil {
		transaction.Metadata["riskScore"] = riskAssessment.RiskScore
		transaction.Metadata["riskLevel"] = riskAssessment.RiskLevel
		transaction.Metadata["riskDecision"] = riskAssessment.Decision
	}

	// Update balances (in production, this should be atomic with transaction creation)
	// TODO: Implement proper balance update in wallet service
	newSenderBalance := senderWallet.Balance - req.Amount
	newReceiverBalance := receiverWallet.Balance + req.Amount

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
		ReceiverAccountID:    transaction.ReceiverAccountID,
		SenderWalletID:       transaction.SenderWalletID,
		ReceiverWalletID:     transaction.ReceiverWalletID,
		Amount:               transaction.Amount,
		Currency:             transaction.Currency,
		TransactionType:      string(transaction.TransactionType),
		TransactionCategory:  string(transaction.TransactionCategory),
		Description:          transaction.Description,
		SenderBalanceAfter:   newSenderBalance,
		ReceiverBalanceAfter: newReceiverBalance,
		Timestamp:            now,
		CompletedAt:          completedTime,
	})

	// Publish money sent event for sender
	producer.ProduceMoneySentEvent(events.MoneySentEvent{
		AccountID:         transaction.SenderAccountID,
		WalletID:          transaction.SenderWalletID,
		TransactionID:     transaction.ID,
		Reference:         transaction.Reference,
		Amount:            transaction.Amount,
		Currency:          transaction.Currency,
		ReceiverAccountID: transaction.ReceiverAccountID,
		ReceiverOmniTag:   req.ReceiverOmniTag,
		Description:       transaction.Description,
		BalanceAfter:      newSenderBalance,
		Timestamp:         completedTime,
	})

	// Publish money received event for receiver
	producer.ProduceMoneyReceivedEvent(events.MoneyReceivedEvent{
		AccountID:       transaction.ReceiverAccountID,
		WalletID:        transaction.ReceiverWalletID,
		TransactionID:   transaction.ID,
		Reference:       transaction.Reference,
		Amount:          transaction.Amount,
		Currency:        transaction.Currency,
		SenderAccountID: transaction.SenderAccountID,
		// SenderOmniTag would need to be fetched if needed
		Description:  transaction.Description,
		BalanceAfter: newReceiverBalance,
		Timestamp:    completedTime,
	})

	// Create response
	response := &models.TransferResponse{
		Status:          "success",
		Message:         "Transfer completed successfully",
		TransactionID:   transaction.ID,
		Reference:       transaction.Reference,
		SenderBalance:   newSenderBalance,
		ReceiverBalance: newReceiverBalance,
		Transaction:     transaction,
	}

	// Store idempotency response
	db.StoreIdempotencyKey(req.IdempotencyKey, senderWallet.AccountID, response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
