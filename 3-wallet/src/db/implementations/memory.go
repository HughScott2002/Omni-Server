package implementations

import (
	"fmt"
	"sync"
	"time"

	"example.com/m/v2/src/models"
)

// MemoryImplementation implements the Database interface using in-memory storage
type MemoryImplementation struct {
	wallets      map[string]*models.Wallet
	virtualCards map[string]*models.VirtualCard
	mu           sync.RWMutex
}

// NewMemoryImplementation creates a new memory implementation
func NewMemoryImplementation() *MemoryImplementation {
	return &MemoryImplementation{
		wallets:      make(map[string]*models.Wallet),
		virtualCards: make(map[string]*models.VirtualCard),
	}
}

func (m *MemoryImplementation) AddWallet(wallet *models.Wallet) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.wallets[wallet.WalletId]; exists {
		return fmt.Errorf("wallet already exists")
	}

	m.wallets[wallet.WalletId] = wallet
	return nil
}

func (m *MemoryImplementation) GetWallet(id string) (*models.Wallet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	wallet, exists := m.wallets[id]
	if !exists {
		return nil, fmt.Errorf("wallet not found")
	}

	return wallet, nil
}
func (m *MemoryImplementation) GetWalletsByAccountId(accountId string) ([]*models.Wallet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var wallets []*models.Wallet
	for _, wallet := range m.wallets {
		if wallet.AccountId == accountId {
			walletCopy := *wallet
			wallets = append(wallets, &walletCopy)
		}
	}
	return wallets, nil
}

func (m *MemoryImplementation) UpdateWallet(wallet *models.Wallet) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.wallets[wallet.WalletId]; !exists {
		return fmt.Errorf("wallet not found")
	}

	wallet.UpdatedAt = time.Now()
	m.wallets[wallet.WalletId] = wallet
	return nil
}

func (m *MemoryImplementation) WalletExists(id string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.wallets[id]
	return exists, nil
}

func (m *MemoryImplementation) DeleteWallet(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.wallets[id]; !exists {
		return fmt.Errorf("wallet not found")
	}

	delete(m.wallets, id)
	return nil
}

func (m *MemoryImplementation) ListWallets(accountId string) ([]*models.Wallet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var wallets []*models.Wallet
	for _, wallet := range m.wallets {
		if wallet.AccountId == accountId {
			wallets = append(wallets, wallet)
		}
	}

	return wallets, nil
}

func (m *MemoryImplementation) UpdateWalletStatus(id string, status models.WalletStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	wallet, exists := m.wallets[id]
	if !exists {
		return fmt.Errorf("wallet not found")
	}

	wallet.Status = status
	wallet.UpdatedAt = time.Now()
	return nil
}

func (m *MemoryImplementation) UpdateWalletBalance(id string, balance float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	wallet, exists := m.wallets[id]
	if !exists {
		return fmt.Errorf("wallet not found")
	}

	wallet.Balance = balance
	wallet.UpdatedAt = time.Now()
	now := time.Now()
	wallet.LastActivity = &now
	return nil
}

func (m *MemoryImplementation) GetDefaultWallet(accountId string) (*models.Wallet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, wallet := range m.wallets {
		if wallet.AccountId == accountId && wallet.IsDefault {
			return wallet, nil
		}
	}

	return nil, fmt.Errorf("no default wallet found")
}

func (m *MemoryImplementation) SetDefaultWallet(accountId string, walletId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify wallet exists and belongs to account
	wallet, exists := m.wallets[walletId]
	if !exists {
		return fmt.Errorf("wallet not found")
	}
	if wallet.AccountId != accountId {
		return fmt.Errorf("wallet does not belong to account")
	}

	// Remove default status from all other wallets for this account
	for _, w := range m.wallets {
		if w.AccountId == accountId {
			w.IsDefault = false
			w.UpdatedAt = time.Now()
		}
	}

	// Set the new default wallet
	wallet.IsDefault = true
	wallet.UpdatedAt = time.Now()
	m.wallets[walletId] = wallet

	return nil
}

func (m *MemoryImplementation) FreezeWallet(accountId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	updated := false
	for _, wallet := range m.wallets {
		if wallet.AccountId == accountId {
			wallet.Status = models.WalletStatusInactive
			wallet.UpdatedAt = time.Now()
			updated = true
		}
	}

	if !updated {
		return fmt.Errorf("no wallets found for account")
	}

	return nil
}

func (m *MemoryImplementation) UnfreezeWallet(accountId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	updated := false
	for _, wallet := range m.wallets {
		if wallet.AccountId == accountId {
			wallet.Status = models.WalletStatusActive
			wallet.UpdatedAt = time.Now()
			updated = true
		}
	}

	if !updated {
		return fmt.Errorf("no wallets found for account")
	}

	return nil
}

// Virtual card operations

func (m *MemoryImplementation) CreateVirtualCard(card *models.VirtualCard) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.virtualCards[card.ID]; exists {
		return fmt.Errorf("virtual card already exists")
	}

	m.virtualCards[card.ID] = card
	return nil
}

func (m *MemoryImplementation) GetVirtualCard(id string) (*models.VirtualCard, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	card, exists := m.virtualCards[id]
	if !exists {
		return nil, fmt.Errorf("virtual card not found")
	}

	return card, nil
}

func (m *MemoryImplementation) GetVirtualCardsByAccountId(accountId string) ([]*models.VirtualCard, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// First, get all wallets for this account
	walletIds := make(map[string]bool)
	for _, wallet := range m.wallets {
		if wallet.AccountId == accountId {
			walletIds[wallet.WalletId] = true
		}
	}

	// Then get all cards for those wallets
	var cards []*models.VirtualCard
	for _, card := range m.virtualCards {
		if walletIds[card.WalletId] {
			cardCopy := *card
			cards = append(cards, &cardCopy)
		}
	}

	return cards, nil
}

// func (m *MemoryImplementation) GetVirtualCardsByWalletId(walletId string) ([]*models.VirtualCard, error) {
// 	m.mu.RLock()
// 	defer m.mu.RUnlock()

// 	var cards []*models.VirtualCard
// 	for _, card := range m.virtualCards {
// 		if card.WalletId == walletId {
// 			cardCopy := *card
// 			cards = append(cards, &cardCopy)
// 		}
// 	}

// 	return cards, nil
// }

func (m *MemoryImplementation) UpdateVirtualCard(card *models.VirtualCard) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.virtualCards[card.ID]; !exists {
		return fmt.Errorf("virtual card not found")
	}

	card.UpdatedAt = time.Now()
	m.virtualCards[card.ID] = card
	return nil
}

func (m *MemoryImplementation) DeleteVirtualCard(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.virtualCards[id]; !exists {
		return fmt.Errorf("virtual card not found")
	}

	delete(m.virtualCards, id)
	return nil
}

func (m *MemoryImplementation) BlockVirtualCard(id string, reason models.CardBlockReason, description string, blockedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	card, exists := m.virtualCards[id]
	if !exists {
		return fmt.Errorf("virtual card not found")
	}

	now := time.Now()
	card.CardStatus = models.VirtualCardStatusBlocked
	card.BlockReason = &reason
	card.BlockReasonDesc = &description
	card.BlockedBy = &blockedBy
	card.BlockedAt = &now
	card.UpdatedAt = now
	m.virtualCards[id] = card

	return nil
}

func (m *MemoryImplementation) TopUpVirtualCard(id string, amount float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	card, exists := m.virtualCards[id]
	if !exists {
		return fmt.Errorf("virtual card not found")
	}

	card.AvailableBalance += amount
	card.TotalToppedUp += amount
	now := time.Now()
	card.LastTopUpDate = &now
	card.UpdatedAt = now
	m.virtualCards[id] = card

	return nil
}

func (m *MemoryImplementation) RequestPhysicalCard(id string, request *models.PhysicalCardRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	card, exists := m.virtualCards[id]
	if !exists {
		return fmt.Errorf("virtual card not found")
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
	m.virtualCards[id] = card

	return nil
}
