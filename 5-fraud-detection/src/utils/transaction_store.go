package utils

import (
	"sync"
	"time"
)

// TransactionHistory stores recent transactions for velocity checks
type TransactionHistory struct {
	TransactionID   string
	SenderAccountID string
	ReceiverAccountID string
	Amount          float64
	Timestamp       time.Time
}

// TransactionStore maintains in-memory transaction history for fraud detection
type TransactionStore struct {
	mu           sync.RWMutex
	transactions []TransactionHistory
	maxAge       time.Duration
	maxSize      int
}

var (
	store *TransactionStore
	once  sync.Once
)

// GetTransactionStore returns the singleton transaction store
func GetTransactionStore() *TransactionStore {
	once.Do(func() {
		store = &TransactionStore{
			transactions: make([]TransactionHistory, 0),
			maxAge:       24 * time.Hour, // Keep 24 hours of history
			maxSize:      10000,           // Keep max 10k transactions
		}
	})
	return store
}

// AddTransaction adds a transaction to the history
func (ts *TransactionStore) AddTransaction(tx TransactionHistory) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.transactions = append(ts.transactions, tx)

	// Clean old transactions if needed
	ts.cleanOldTransactions()
}

// cleanOldTransactions removes transactions older than maxAge or exceeds maxSize
func (ts *TransactionStore) cleanOldTransactions() {
	cutoff := time.Now().Add(-ts.maxAge)
	validTransactions := make([]TransactionHistory, 0)

	for _, tx := range ts.transactions {
		if tx.Timestamp.After(cutoff) {
			validTransactions = append(validTransactions, tx)
		}
	}

	ts.transactions = validTransactions

	// If still too many, keep only the most recent
	if len(ts.transactions) > ts.maxSize {
		ts.transactions = ts.transactions[len(ts.transactions)-ts.maxSize:]
	}
}

// GetRecentTransactionsBySender returns transactions by sender in the last duration
func (ts *TransactionStore) GetRecentTransactionsBySender(senderID string, duration time.Duration) []TransactionHistory {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	result := make([]TransactionHistory, 0)

	for _, tx := range ts.transactions {
		if tx.SenderAccountID == senderID && tx.Timestamp.After(cutoff) {
			result = append(result, tx)
		}
	}

	return result
}

// GetRecentTransactionsByReceiver returns transactions to receiver in the last duration
func (ts *TransactionStore) GetRecentTransactionsByReceiver(receiverID string, duration time.Duration) []TransactionHistory {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	result := make([]TransactionHistory, 0)

	for _, tx := range ts.transactions {
		if tx.ReceiverAccountID == receiverID && tx.Timestamp.After(cutoff) {
			result = append(result, tx)
		}
	}

	return result
}

// GetRecentTransactionsBetween returns transactions between sender and receiver
func (ts *TransactionStore) GetRecentTransactionsBetween(senderID, receiverID string, duration time.Duration) []TransactionHistory {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	result := make([]TransactionHistory, 0)

	for _, tx := range ts.transactions {
		if tx.SenderAccountID == senderID && tx.ReceiverAccountID == receiverID && tx.Timestamp.After(cutoff) {
			result = append(result, tx)
		}
	}

	return result
}

// GetTotalAmountBySender calculates total amount sent by sender in the last duration
func (ts *TransactionStore) GetTotalAmountBySender(senderID string, duration time.Duration) float64 {
	transactions := ts.GetRecentTransactionsBySender(senderID, duration)
	total := 0.0
	for _, tx := range transactions {
		total += tx.Amount
	}
	return total
}

// CountTransactionsBySender counts transactions by sender in the last duration
func (ts *TransactionStore) CountTransactionsBySender(senderID string, duration time.Duration) int {
	return len(ts.GetRecentTransactionsBySender(senderID, duration))
}
