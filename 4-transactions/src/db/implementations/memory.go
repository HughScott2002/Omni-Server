package implementations

import (
	"fmt"
	"sync"
	"time"

	"example.com/transactions/v1/src/models"
)

type MemoryDB struct {
	transactions    map[string]*models.Transaction
	references      map[string]string // reference -> transaction ID
	accountTxs      map[string][]string
	walletTxs       map[string][]string
	idempotencyKeys map[string]interface{}
	mu              sync.RWMutex
}

func NewMemoryImplementation() *MemoryDB {
	return &MemoryDB{
		transactions:    make(map[string]*models.Transaction),
		references:      make(map[string]string),
		accountTxs:      make(map[string][]string),
		walletTxs:       make(map[string][]string),
		idempotencyKeys: make(map[string]interface{}),
	}
}

// Transaction operations

func (m *MemoryDB) CreateTransaction(tx *models.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.transactions[tx.ID] = tx
	m.references[tx.Reference] = tx.ID

	// Add to account's transaction list
	if tx.SenderAccountID != "" {
		m.accountTxs[tx.SenderAccountID] = append(m.accountTxs[tx.SenderAccountID], tx.ID)
	}
	if tx.ReceiverAccountID != "" {
		m.accountTxs[tx.ReceiverAccountID] = append(m.accountTxs[tx.ReceiverAccountID], tx.ID)
	}

	// Add to wallet's transaction list
	if tx.SenderWalletID != "" {
		m.walletTxs[tx.SenderWalletID] = append(m.walletTxs[tx.SenderWalletID], tx.ID)
	}
	if tx.ReceiverWalletID != "" {
		m.walletTxs[tx.ReceiverWalletID] = append(m.walletTxs[tx.ReceiverWalletID], tx.ID)
	}

	return nil
}

func (m *MemoryDB) GetTransaction(id string) (*models.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tx, exists := m.transactions[id]
	if !exists {
		return nil, fmt.Errorf("transaction not found")
	}

	return tx, nil
}

func (m *MemoryDB) GetTransactionByReference(reference string) (*models.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, exists := m.references[reference]
	if !exists {
		return nil, fmt.Errorf("transaction not found")
	}

	return m.transactions[id], nil
}

func (m *MemoryDB) UpdateTransaction(tx *models.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.transactions[tx.ID]; !exists {
		return fmt.Errorf("transaction not found")
	}

	m.transactions[tx.ID] = tx
	return nil
}

func (m *MemoryDB) UpdateTransactionStatus(id string, status models.TransactionStatus, failedReason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tx, exists := m.transactions[id]
	if !exists {
		return fmt.Errorf("transaction not found")
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

	return nil
}

func (m *MemoryDB) GetTransactionsByAccountID(accountID string, params *models.TransactionHistoryParams) ([]*models.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids, exists := m.accountTxs[accountID]
	if !exists {
		return []*models.Transaction{}, nil
	}

	transactions := make([]*models.Transaction, 0)
	for _, id := range ids {
		tx := m.transactions[id]
		if tx == nil {
			continue
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

	// Apply pagination
	if params != nil {
		start := params.Offset
		end := params.Offset + params.Limit

		if start > len(transactions) {
			return []*models.Transaction{}, nil
		}
		if end > len(transactions) {
			end = len(transactions)
		}

		transactions = transactions[start:end]
	}

	return transactions, nil
}

func (m *MemoryDB) GetTransactionsByWalletID(walletID string, params *models.TransactionHistoryParams) ([]*models.Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids, exists := m.walletTxs[walletID]
	if !exists {
		return []*models.Transaction{}, nil
	}

	transactions := make([]*models.Transaction, 0)
	for _, id := range ids {
		tx := m.transactions[id]
		if tx == nil {
			continue
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

	// Apply pagination
	if params != nil {
		start := params.Offset
		end := params.Offset + params.Limit

		if start > len(transactions) {
			return []*models.Transaction{}, nil
		}
		if end > len(transactions) {
			end = len(transactions)
		}

		transactions = transactions[start:end]
	}

	return transactions, nil
}

// Idempotency operations

func (m *MemoryDB) StoreIdempotencyKey(key string, accountID string, response interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idempotencyKey := fmt.Sprintf("%s:%s", accountID, key)
	m.idempotencyKeys[idempotencyKey] = response

	return nil
}

func (m *MemoryDB) GetIdempotencyResponse(key string, accountID string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	idempotencyKey := fmt.Sprintf("%s:%s", accountID, key)
	response, exists := m.idempotencyKeys[idempotencyKey]
	if !exists {
		return nil, nil
	}

	return response, nil
}

func (m *MemoryDB) DeleteIdempotencyKey(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.idempotencyKeys, key)
	return nil
}
