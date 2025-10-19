package implementations

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"example.com/transactions/v1/src/models"
	"github.com/redis/go-redis/v9"
)

type RedisDB struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisImplementation(client *redis.Client) *RedisDB {
	return &RedisDB{
		client: client,
		ctx:    context.Background(),
	}
}

// Transaction operations

func (r *RedisDB) CreateTransaction(tx *models.Transaction) error {
	key := fmt.Sprintf("transaction:%s", tx.ID)
	refKey := fmt.Sprintf("transaction:ref:%s", tx.Reference)

	// Marshal transaction to JSON
	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %v", err)
	}

	// Store transaction by ID
	if err := r.client.Set(r.ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to store transaction: %v", err)
	}

	// Store reference mapping
	if err := r.client.Set(r.ctx, refKey, tx.ID, 0).Err(); err != nil {
		return fmt.Errorf("failed to store transaction reference: %v", err)
	}

	// Add to account's transaction list
	if tx.SenderAccountID != "" {
		senderKey := fmt.Sprintf("account:%s:transactions", tx.SenderAccountID)
		r.client.ZAdd(r.ctx, senderKey, redis.Z{
			Score:  float64(tx.CreatedAt.Unix()),
			Member: tx.ID,
		})
	}

	if tx.ReceiverAccountID != "" {
		receiverKey := fmt.Sprintf("account:%s:transactions", tx.ReceiverAccountID)
		r.client.ZAdd(r.ctx, receiverKey, redis.Z{
			Score:  float64(tx.CreatedAt.Unix()),
			Member: tx.ID,
		})
	}

	// Add to wallet's transaction list
	if tx.SenderWalletID != "" {
		walletKey := fmt.Sprintf("wallet:%s:transactions", tx.SenderWalletID)
		r.client.ZAdd(r.ctx, walletKey, redis.Z{
			Score:  float64(tx.CreatedAt.Unix()),
			Member: tx.ID,
		})
	}

	if tx.ReceiverWalletID != "" {
		walletKey := fmt.Sprintf("wallet:%s:transactions", tx.ReceiverWalletID)
		r.client.ZAdd(r.ctx, walletKey, redis.Z{
			Score:  float64(tx.CreatedAt.Unix()),
			Member: tx.ID,
		})
	}

	return nil
}

func (r *RedisDB) GetTransaction(id string) (*models.Transaction, error) {
	key := fmt.Sprintf("transaction:%s", id)

	data, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("transaction not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %v", err)
	}

	var tx models.Transaction
	if err := json.Unmarshal([]byte(data), &tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %v", err)
	}

	return &tx, nil
}

func (r *RedisDB) GetTransactionByReference(reference string) (*models.Transaction, error) {
	refKey := fmt.Sprintf("transaction:ref:%s", reference)

	// Get transaction ID from reference
	id, err := r.client.Get(r.ctx, refKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("transaction not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction reference: %v", err)
	}

	return r.GetTransaction(id)
}

func (r *RedisDB) UpdateTransaction(tx *models.Transaction) error {
	key := fmt.Sprintf("transaction:%s", tx.ID)

	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %v", err)
	}

	if err := r.client.Set(r.ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to update transaction: %v", err)
	}

	return nil
}

func (r *RedisDB) UpdateTransactionStatus(id string, status models.TransactionStatus, failedReason string) error {
	tx, err := r.GetTransaction(id)
	if err != nil {
		return err
	}

	tx.Status = status
	tx.UpdatedAt = time.Now()

	if status == models.TransactionStatusCompleted {
		now := time.Now()
		tx.CompletedAt = &now
	}

	if failedReason != "" {
		tx.FailedReason = failedReason
	}

	return r.UpdateTransaction(tx)
}

func (r *RedisDB) GetTransactionsByAccountID(accountID string, params *models.TransactionHistoryParams) ([]*models.Transaction, error) {
	key := fmt.Sprintf("account:%s:transactions", accountID)

	// Get transaction IDs from sorted set (most recent first)
	start := int64(params.Offset)
	stop := int64(params.Offset + params.Limit - 1)

	ids, err := r.client.ZRevRange(r.ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction IDs: %v", err)
	}

	transactions := make([]*models.Transaction, 0, len(ids))
	for _, id := range ids {
		tx, err := r.GetTransaction(id)
		if err != nil {
			continue // Skip if transaction not found
		}

		// Apply filters if specified
		if params != nil {
			if params.TransactionType != "" && tx.TransactionType != params.TransactionType {
				continue
			}
			if params.TransactionCategory != "" && tx.TransactionCategory != params.TransactionCategory {
				continue
			}
			if params.Status != "" && tx.Status != params.Status {
				continue
			}
			if params.MinAmount > 0 && tx.Amount < params.MinAmount {
				continue
			}
			if params.MaxAmount > 0 && tx.Amount > params.MaxAmount {
				continue
			}
			if params.StartDate != nil && tx.CreatedAt.Before(*params.StartDate) {
				continue
			}
			if params.EndDate != nil && tx.CreatedAt.After(*params.EndDate) {
				continue
			}
		}

		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func (r *RedisDB) GetTransactionsByWalletID(walletID string, params *models.TransactionHistoryParams) ([]*models.Transaction, error) {
	key := fmt.Sprintf("wallet:%s:transactions", walletID)

	// Get transaction IDs from sorted set (most recent first)
	start := int64(params.Offset)
	stop := int64(params.Offset + params.Limit - 1)

	ids, err := r.client.ZRevRange(r.ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction IDs: %v", err)
	}

	transactions := make([]*models.Transaction, 0, len(ids))
	for _, id := range ids {
		tx, err := r.GetTransaction(id)
		if err != nil {
			continue // Skip if transaction not found
		}

		// Apply filters if specified
		if params != nil {
			if params.TransactionType != "" && tx.TransactionType != params.TransactionType {
				continue
			}
			if params.TransactionCategory != "" && tx.TransactionCategory != params.TransactionCategory {
				continue
			}
			if params.Status != "" && tx.Status != params.Status {
				continue
			}
			if params.MinAmount > 0 && tx.Amount < params.MinAmount {
				continue
			}
			if params.MaxAmount > 0 && tx.Amount > params.MaxAmount {
				continue
			}
			if params.StartDate != nil && tx.CreatedAt.Before(*params.StartDate) {
				continue
			}
			if params.EndDate != nil && tx.CreatedAt.After(*params.EndDate) {
				continue
			}
		}

		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// Idempotency operations

func (r *RedisDB) StoreIdempotencyKey(key string, accountID string, response interface{}) error {
	idempotencyKey := fmt.Sprintf("idempotency:%s:%s", accountID, key)

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal idempotency response: %v", err)
	}

	// Store with 24 hour expiry
	if err := r.client.Set(r.ctx, idempotencyKey, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store idempotency key: %v", err)
	}

	return nil
}

func (r *RedisDB) GetIdempotencyResponse(key string, accountID string) (interface{}, error) {
	idempotencyKey := fmt.Sprintf("idempotency:%s:%s", accountID, key)

	data, err := r.client.Get(r.ctx, idempotencyKey).Result()
	if err == redis.Nil {
		return nil, nil // Not found, not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get idempotency key: %v", err)
	}

	var response interface{}
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal idempotency response: %v", err)
	}

	return response, nil
}

func (r *RedisDB) DeleteIdempotencyKey(key string) error {
	// Note: key should already include account ID
	if err := r.client.Del(r.ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete idempotency key: %v", err)
	}

	return nil
}
