package db

import (
	"fmt"
	"log"
	"os"
	"strings"

	"example.com/m/v2/src/db/implementations"
	"example.com/m/v2/src/models"
)

// Global database instance
var db Database

// Database interface defines all operations that must be implemented by any database implementation
type Database interface {
	// Wallet operations
	AddWallet(wallet *models.Wallet) error
	GetWallet(id string) (*models.Wallet, error)
	GetWalletsByAccountId(accountId string) ([]*models.Wallet, error)
	UpdateWallet(wallet *models.Wallet) error
	WalletExists(id string) (bool, error)
	DeleteWallet(id string) error
	ListWallets(accountId string) ([]*models.Wallet, error)
	UpdateWalletStatus(id string, status models.WalletStatus) error
	UpdateWalletBalance(id string, balance float64) error
	GetDefaultWallet(accountId string) (*models.Wallet, error)
	SetDefaultWallet(accountId string, walletId string) error
	FreezeWallet(accountId string) error
	UnfreezeWallet(accountId string) error

	// Virtual card operations
	CreateVirtualCard(card *models.VirtualCard) error
	GetVirtualCard(id string) (*models.VirtualCard, error)
	GetVirtualCardsByAccountId(accountId string) ([]*models.VirtualCard, error)
	// GetVirtualCardsByWalletId(walletId string) ([]*models.VirtualCard, error)
	UpdateVirtualCard(card *models.VirtualCard) error
	DeleteVirtualCard(id string) error
	BlockVirtualCard(id string, reason models.CardBlockReason, description string, blockedBy string) error
	TopUpVirtualCard(id string, amount float64) error
	RequestPhysicalCard(id string, request *models.PhysicalCardRequest) error
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
			// panic("Failed to connect to Redis: " + err.Error())
			return fmt.Errorf("Failed to connect to Redis: " + err.Error())
		}
		db = implementations.RedisImplementation(RedisClient)
		log.Println("USING REDIS IN WALLET SERVICE")
	case env == "prod" || env == "production":
		return fmt.Errorf("production database not implemented")
	default:
		return fmt.Errorf("unsupported environment or mode")
	}

	return nil
}

// Helper functions that use the database interface

func AddWallet(wallet *models.Wallet) error {
	exists, err := db.WalletExists(wallet.WalletId)
	if err != nil {
		return fmt.Errorf("error checking wallet existence: %v", err)
	}
	if exists {
		return fmt.Errorf("wallet already exists")
	}
	return db.AddWallet(wallet)
}

func GetWallet(id string) (*models.Wallet, error) {
	return db.GetWallet(id)
}
func GetWalletsByAccountId(accountId string) ([]*models.Wallet, error) {
	return db.GetWalletsByAccountId(accountId)
}
func UpdateWallet(wallet *models.Wallet) error {
	return db.UpdateWallet(wallet)
}

func WalletExists(id string) (bool, error) {
	return db.WalletExists(id)
}

func DeleteWallet(id string) error {
	return db.DeleteWallet(id)
}

func ListWallets(accountId string) ([]*models.Wallet, error) {
	return db.ListWallets(accountId)
}

func UpdateWalletStatus(id string, status models.WalletStatus) error {
	return db.UpdateWalletStatus(id, status)
}

func UpdateWalletBalance(id string, balance float64) error {
	return db.UpdateWalletBalance(id, balance)
}

func GetDefaultWallet(accountId string) (*models.Wallet, error) {
	return db.GetDefaultWallet(accountId)
}

func SetDefaultWallet(accountId string, walletId string) error {
	return db.SetDefaultWallet(accountId, walletId)
}

func FreezeWallet(accountId string) error {
	return db.FreezeWallet(accountId)
}

func UnfreezeWallet(accountId string) error {
	return db.UnfreezeWallet(accountId)
}

// Virtual card helper functions

func CreateVirtualCard(card *models.VirtualCard) error {
	return db.CreateVirtualCard(card)
}

func GetVirtualCard(id string) (*models.VirtualCard, error) {
	return db.GetVirtualCard(id)
}

func GetVirtualCardsByAccountId(accountId string) ([]*models.VirtualCard, error) {
	return db.GetVirtualCardsByAccountId(accountId)
}

// func GetVirtualCardsByWalletId(walletId string) ([]*models.VirtualCard, error) {
// 	return db.GetVirtualCardsByWalletId(walletId)
// }

func UpdateVirtualCard(card *models.VirtualCard) error {
	return db.UpdateVirtualCard(card)
}

func DeleteVirtualCard(id string) error {
	return db.DeleteVirtualCard(id)
}

func BlockVirtualCard(id string, reason models.CardBlockReason, description string, blockedBy string) error {
	return db.BlockVirtualCard(id, reason, description, blockedBy)
}

func TopUpVirtualCard(id string, amount float64) error {
	return db.TopUpVirtualCard(id, amount)
}

func RequestPhysicalCard(id string, request *models.PhysicalCardRequest) error {
	return db.RequestPhysicalCard(id, request)
}
