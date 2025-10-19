package db

import (
	"fmt"
	"log"
	"os"
	"strings"

	"example.com/transactions/v1/src/db/implementations"
	"example.com/transactions/v1/src/models"
)

// Global database instance
var db Database

// Database interface defines all operations that must be implemented by any database implementation
type Database interface {
	// Transaction operations
	CreateTransaction(tx *models.Transaction) error
	GetTransaction(id string) (*models.Transaction, error)
	GetTransactionByReference(reference string) (*models.Transaction, error)
	UpdateTransaction(tx *models.Transaction) error
	UpdateTransactionStatus(id string, status models.TransactionStatus, failedReason string) error
	GetTransactionsByAccountID(accountID string, params *models.TransactionHistoryParams) ([]*models.Transaction, error)
	GetTransactionsByWalletID(walletID string, params *models.TransactionHistoryParams) ([]*models.Transaction, error)

	// Idempotency operations
	StoreIdempotencyKey(key string, accountID string, response interface{}) error
	GetIdempotencyResponse(key string, accountID string) (interface{}, error)
	DeleteIdempotencyKey(key string) error
}

// Init initializes the appropriate database implementation based on environment variables
func Init() error {
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	mode := strings.ToLower(os.Getenv("MODE"))

	switch {
	case env == "local" && mode == "memcached":
		db = implementations.NewMemoryImplementation()
	case env == "local" && mode != "redis":
		RedisClient, err := InitRedis()
		if err != nil {
			return fmt.Errorf("failed to connect to Redis: %v", err)
		}
		db = implementations.NewRedisImplementation(RedisClient)
		log.Println("USING REDIS IN TRANSACTION SERVICE")
	case env == "prod" || env == "production":
		return fmt.Errorf("production database not implemented")
	default:
		return fmt.Errorf("unsupported environment or mode")
	}

	return nil
}

// Helper functions that use the database interface

func CreateTransaction(tx *models.Transaction) error {
	return db.CreateTransaction(tx)
}

func GetTransaction(id string) (*models.Transaction, error) {
	return db.GetTransaction(id)
}

func GetTransactionByReference(reference string) (*models.Transaction, error) {
	return db.GetTransactionByReference(reference)
}

func UpdateTransaction(tx *models.Transaction) error {
	return db.UpdateTransaction(tx)
}

func UpdateTransactionStatus(id string, status models.TransactionStatus, failedReason string) error {
	return db.UpdateTransactionStatus(id, status, failedReason)
}

func GetTransactionsByAccountID(accountID string, params *models.TransactionHistoryParams) ([]*models.Transaction, error) {
	return db.GetTransactionsByAccountID(accountID, params)
}

func GetTransactionsByWalletID(walletID string, params *models.TransactionHistoryParams) ([]*models.Transaction, error) {
	return db.GetTransactionsByWalletID(walletID, params)
}

func StoreIdempotencyKey(key string, accountID string, response interface{}) error {
	return db.StoreIdempotencyKey(key, accountID, response)
}

func GetIdempotencyResponse(key string, accountID string) (interface{}, error) {
	return db.GetIdempotencyResponse(key, accountID)
}

func DeleteIdempotencyKey(key string) error {
	return db.DeleteIdempotencyKey(key)
}
