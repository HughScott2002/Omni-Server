package implementations

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"example.com/m/v2/src/models"
	"github.com/go-redis/redis/v8"
)

// Key patterns for Redis
const (
	walletKey         = "wallet:%s"                // wallet:{walletId}
	accountWalletsKey = "account:wallets:%s"       // account:wallets:{accountId}
	defaultWalletKey  = "account:default:%s"       // account:default:{accountId}
	virtualCardKey    = "virtual_card:%s"          // virtual_card:{cardId}
	accountCardsKey   = "account:virtual_cards:%s" // account:virtual_cards:{accountId}
)

// Redis extends db.RedisDB with the actual Redis client
type Redis struct {
	client *redis.Client
}

// RedisImplementation creates a new Redis implementation
func RedisImplementation(client *redis.Client) *Redis {
	return &Redis{
		client: client,
	}
}

// RedisDB implementation
func (r *Redis) AddWallet(wallet *models.Wallet) error {
	ctx := context.Background()

	exists, err := r.WalletExists(wallet.WalletId)
	if err != nil {
		return fmt.Errorf("error checking wallet existence: %v", err)
	}
	if exists {
		return fmt.Errorf("wallet already exists")
	}

	data, err := json.Marshal(wallet)
	if err != nil {
		return fmt.Errorf("error marshaling wallet: %v", err)
	}

	pipe := r.client.Pipeline()

	// Store wallet data
	pipe.Set(ctx, fmt.Sprintf(walletKey, wallet.WalletId), data, 0)

	// Add to account's wallet list
	pipe.SAdd(ctx, fmt.Sprintf(accountWalletsKey, wallet.AccountId), wallet.WalletId)

	// If this is a default wallet, set it
	if wallet.IsDefault {
		pipe.Set(ctx, fmt.Sprintf(defaultWalletKey, wallet.AccountId), wallet.WalletId, 0)
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (r *Redis) GetWallet(id string) (*models.Wallet, error) {
	ctx := context.Background()

	data, err := r.client.Get(ctx, fmt.Sprintf(walletKey, id)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("wallet not found")
		}
		return nil, err
	}

	var wallet models.Wallet
	if err := json.Unmarshal(data, &wallet); err != nil {
		return nil, fmt.Errorf("error unmarshaling wallet: %v", err)
	}

	return &wallet, nil
}

func (r *Redis) GetWalletsByAccountId(accountId string) ([]*models.Wallet, error) {
	pattern := fmt.Sprintf("wallet:account:%s:*", accountId)
	ctx := context.Background()

	var wallets []*models.Wallet
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		walletJSON, err := r.client.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}

		var wallet models.Wallet
		if err := json.Unmarshal(walletJSON, &wallet); err != nil {
			continue
		}

		wallets = append(wallets, &wallet)
	}

	return wallets, nil
}

func (r *Redis) UpdateWallet(wallet *models.Wallet) error {
	ctx := context.Background()

	exists, err := r.WalletExists(wallet.WalletId)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("wallet not found")
	}

	wallet.UpdatedAt = time.Now()

	data, err := json.Marshal(wallet)
	if err != nil {
		return fmt.Errorf("error marshaling wallet: %v", err)
	}

	pipe := r.client.Pipeline()

	// Update wallet data
	pipe.Set(ctx, fmt.Sprintf(walletKey, wallet.WalletId), data, 0)

	// Handle default wallet status changes
	if wallet.IsDefault {
		pipe.Set(ctx, fmt.Sprintf(defaultWalletKey, wallet.AccountId), wallet.WalletId, 0)
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (r *Redis) WalletExists(id string) (bool, error) {
	ctx := context.Background()
	exists, err := r.client.Exists(ctx, fmt.Sprintf(walletKey, id)).Result()
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}

func (r *Redis) DeleteWallet(id string) error {
	ctx := context.Background()

	wallet, err := r.GetWallet(id)
	if err != nil {
		return err
	}

	pipe := r.client.Pipeline()

	// Remove wallet data
	pipe.Del(ctx, fmt.Sprintf(walletKey, id))

	// Remove from account's wallet list
	pipe.SRem(ctx, fmt.Sprintf(accountWalletsKey, wallet.AccountId), id)

	// If this was the default wallet, remove that reference
	if wallet.IsDefault {
		pipe.Del(ctx, fmt.Sprintf(defaultWalletKey, wallet.AccountId))
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (r *Redis) ListWallets(accountId string) ([]*models.Wallet, error) {
	ctx := context.Background()

	// Get all wallet IDs for this account
	walletIds, err := r.client.SMembers(ctx, fmt.Sprintf(accountWalletsKey, accountId)).Result()
	if err != nil {
		return nil, err
	}

	wallets := make([]*models.Wallet, 0, len(walletIds))
	for _, id := range walletIds {
		wallet, err := r.GetWallet(id)
		if err != nil {
			continue // Skip failed wallet retrievals
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}

func (r *Redis) UpdateWalletStatus(id string, status models.WalletStatus) error {
	wallet, err := r.GetWallet(id)
	if err != nil {
		return err
	}

	wallet.Status = status
	wallet.UpdatedAt = time.Now()

	return r.UpdateWallet(wallet)
}

func (r *Redis) UpdateWalletBalance(id string, balance float64) error {
	wallet, err := r.GetWallet(id)
	if err != nil {
		return err
	}

	wallet.Balance = balance
	wallet.UpdatedAt = time.Now()
	now := time.Now()
	wallet.LastActivity = &now

	return r.UpdateWallet(wallet)
}

func (r *Redis) GetDefaultWallet(accountId string) (*models.Wallet, error) {
	ctx := context.Background()

	walletId, err := r.client.Get(ctx, fmt.Sprintf(defaultWalletKey, accountId)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("no default wallet found")
		}
		return nil, err
	}

	return r.GetWallet(walletId)
}

func (r *Redis) SetDefaultWallet(accountId string, walletId string) error {
	ctx := context.Background()

	// First, get the wallet to make sure it exists and belongs to the account
	wallet, err := r.GetWallet(walletId)
	if err != nil {
		return err
	}
	if wallet.AccountId != accountId {
		return fmt.Errorf("wallet does not belong to account")
	}

	pipe := r.client.Pipeline()

	// Update the current wallet to be the default
	wallet.IsDefault = true
	wallet.UpdatedAt = time.Now()
	data, err := json.Marshal(wallet)
	if err != nil {
		return err
	}
	pipe.Set(ctx, fmt.Sprintf(walletKey, walletId), data, 0)

	// Set as the default wallet for the account
	pipe.Set(ctx, fmt.Sprintf(defaultWalletKey, accountId), walletId, 0)

	// Remove default status from other wallets
	otherWallets, err := r.ListWallets(accountId)
	if err != nil {
		return err
	}

	for _, w := range otherWallets {
		if w.WalletId != walletId && w.IsDefault {
			w.IsDefault = false
			w.UpdatedAt = time.Now()
			data, err := json.Marshal(w)
			if err != nil {
				continue
			}
			pipe.Set(ctx, fmt.Sprintf(walletKey, w.WalletId), data, 0)
		}
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (r *Redis) FreezeWallet(accountId string) error {
	wallets, err := r.ListWallets(accountId)
	if err != nil {
		return err
	}

	ctx := context.Background()
	pipe := r.client.Pipeline()

	for _, wallet := range wallets {
		wallet.Status = models.WalletStatusInactive
		wallet.UpdatedAt = time.Now()

		data, err := json.Marshal(wallet)
		if err != nil {
			continue
		}
		pipe.Set(ctx, fmt.Sprintf(walletKey, wallet.WalletId), data, 0)
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (r *Redis) UnfreezeWallet(accountId string) error {
	wallets, err := r.ListWallets(accountId)
	if err != nil {
		return err
	}

	ctx := context.Background()
	pipe := r.client.Pipeline()

	for _, wallet := range wallets {
		wallet.Status = models.WalletStatusActive
		wallet.UpdatedAt = time.Now()

		data, err := json.Marshal(wallet)
		if err != nil {
			continue
		}
		pipe.Set(ctx, fmt.Sprintf(walletKey, wallet.WalletId), data, 0)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// Virtual card operations

func (r *Redis) CreateVirtualCard(card *models.VirtualCard) error {
	ctx := context.Background()

	data, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("error marshaling virtual card: %v", err)
	}

	pipe := r.client.Pipeline()

	// Store card data
	pipe.Set(ctx, fmt.Sprintf(virtualCardKey, card.ID), data, 0)

	// Add to account's card list (via wallet)
	wallet, err := r.GetWallet(card.WalletId)
	if err != nil {
		return fmt.Errorf("error getting wallet: %v", err)
	}
	pipe.SAdd(ctx, fmt.Sprintf(accountCardsKey, wallet.AccountId), card.ID)

	_, err = pipe.Exec(ctx)
	return err
}

func (r *Redis) GetVirtualCard(id string) (*models.VirtualCard, error) {
	ctx := context.Background()

	data, err := r.client.Get(ctx, fmt.Sprintf(virtualCardKey, id)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("virtual card not found")
		}
		return nil, err
	}

	var card models.VirtualCard
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("error unmarshaling virtual card: %v", err)
	}

	return &card, nil
}

func (r *Redis) GetVirtualCardsByAccountId(accountId string) ([]*models.VirtualCard, error) {
	ctx := context.Background()

	// Get all card IDs for this account
	cardIds, err := r.client.SMembers(ctx, fmt.Sprintf(accountCardsKey, accountId)).Result()
	if err != nil {
		return nil, err
	}

	cards := make([]*models.VirtualCard, 0, len(cardIds))
	for _, id := range cardIds {
		card, err := r.GetVirtualCard(id)
		if err != nil {
			continue // Skip failed card retrievals
		}
		cards = append(cards, card)
	}

	return cards, nil
}

// func (r *Redis) GetVirtualCardsByWalletId(walletId string) ([]*models.VirtualCard, error) {
// 	ctx := context.Background()

// 	// Get wallet to find account ID
// 	wallet, err := r.GetWallet(walletId)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Get all card IDs for this account
// 	cardIds, err := r.client.SMembers(ctx, fmt.Sprintf(accountCardsKey, wallet.AccountId)).Result()
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Filter cards by wallet ID
// 	cards := make([]*models.VirtualCard, 0)
// 	for _, id := range cardIds {
// 		card, err := r.GetVirtualCard(id)
// 		if err != nil {
// 			continue // Skip failed card retrievals
// 		}
// 		if card.WalletId == walletId {
// 			cards = append(cards, card)
// 		}
// 	}
// 	return cards, nil
// }

func (r *Redis) UpdateVirtualCard(card *models.VirtualCard) error {
	ctx := context.Background()

	card.UpdatedAt = time.Now()

	data, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("error marshaling virtual card: %v", err)
	}

	return r.client.Set(ctx, fmt.Sprintf(virtualCardKey, card.ID), data, 0).Err()
}

func (r *Redis) DeleteVirtualCard(id string) error {
	ctx := context.Background()

	card, err := r.GetVirtualCard(id)
	if err != nil {
		return err
	}

	wallet, err := r.GetWallet(card.WalletId)
	if err != nil {
		return err
	}

	pipe := r.client.Pipeline()

	// Remove card data
	pipe.Del(ctx, fmt.Sprintf(virtualCardKey, id))

	// Remove from account's card list
	pipe.SRem(ctx, fmt.Sprintf(accountCardsKey, wallet.AccountId), id)

	_, err = pipe.Exec(ctx)
	return err
}

func (r *Redis) BlockVirtualCard(id string, reason models.CardBlockReason, description string, blockedBy string) error {
	card, err := r.GetVirtualCard(id)
	if err != nil {
		return err
	}

	now := time.Now()
	card.CardStatus = models.VirtualCardStatusBlocked
	card.BlockReason = &reason
	card.BlockReasonDesc = &description
	card.BlockedBy = &blockedBy
	card.BlockedAt = &now
	card.UpdatedAt = now

	return r.UpdateVirtualCard(card)
}

func (r *Redis) TopUpVirtualCard(id string, amount float64) error {
	card, err := r.GetVirtualCard(id)
	if err != nil {
		return err
	}

	card.AvailableBalance += amount
	card.TotalToppedUp += amount
	now := time.Now()
	card.LastTopUpDate = &now
	card.UpdatedAt = now

	return r.UpdateVirtualCard(card)
}

func (r *Redis) RequestPhysicalCard(id string, request *models.PhysicalCardRequest) error {
	card, err := r.GetVirtualCard(id)
	if err != nil {
		return err
	}

	now := time.Now()
	status := "pending"
	card.IsPhysicalCardRequest = true
	card.PhysicalCardReqAt = &now
	card.DeliveryAddress = &request.DeliveryAddress
	card.DeliveryCity = &request.DeliveryCity
	card.DeliveryCountry = &request.DeliveryCountry
	card.DeliveryPostalCode = &request.DeliveryPostalCode
	card.PhysicalCardStatus = &status
	card.UpdatedAt = now

	return r.UpdateVirtualCard(card)
}
