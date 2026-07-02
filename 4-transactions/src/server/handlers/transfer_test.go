package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"example.com/transactions/v1/src/db"
	"example.com/transactions/v1/src/models"
	"example.com/transactions/v1/src/server/handlers"
	"github.com/go-chi/chi/v5"
)

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "local")
	os.Setenv("MODE", "memcached")
	os.Setenv("KAFKA_DISABLED", "true")
	os.Exit(m.Run())
}

type fakeWallet struct {
	WalletID  string
	AccountID string
	Balance   float64
	Status    string
	IsDefault bool
}

// fakeBackend stands in for the user, wallet, and fraud-detection services.
type fakeBackend struct {
	mu              sync.Mutex
	wallets         map[string]*fakeWallet
	omniTags        map[string]string // omniTag -> accountID
	declineTransfer bool              // wallet service returns 400 on transfer
	failTransfer    bool              // wallet service returns 500 on transfer
	transferCalls   []string          // "from|to|reference"
	applied         map[string]bool   // references already applied (wallet-side idempotency)
}

func (f *fakeBackend) walletJSON(w *fakeWallet) map[string]interface{} {
	return map[string]interface{}{
		"walletId":  w.WalletID,
		"accountId": w.AccountID,
		"balance":   w.Balance,
		"currency":  "USD",
		"status":    w.Status,
		"isDefault": w.IsDefault,
	}
}

func (f *fakeBackend) start(t *testing.T) {
	t.Helper()

	wallets := chi.NewRouter()
	wallets.Get("/api/wallets/list/{accountId}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		accountID := chi.URLParam(r, "accountId")
		list := []map[string]interface{}{}
		for _, wal := range f.wallets {
			if wal.AccountID == accountID {
				list = append(list, f.walletJSON(wal))
			}
		}
		json.NewEncoder(w).Encode(list)
	})
	wallets.Get("/api/wallets/{walletId}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		wal, ok := f.wallets[chi.URLParam(r, "walletId")]
		if !ok {
			http.Error(w, "Wallet not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(f.walletJSON(wal))
	})
	wallets.Post("/api/wallets/transfer", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		var req struct {
			FromWalletId string  `json:"fromWalletId"`
			ToWalletId   string  `json:"toWalletId"`
			Amount       float64 `json:"amount"`
			Reference    string  `json:"reference"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		f.transferCalls = append(f.transferCalls, req.FromWalletId+"|"+req.ToWalletId+"|"+req.Reference)
		if f.failTransfer {
			http.Error(w, "wallet service exploded", http.StatusInternalServerError)
			return
		}
		from, okFrom := f.wallets[req.FromWalletId]
		to, okTo := f.wallets[req.ToWalletId]
		if !okFrom || !okTo {
			http.Error(w, "wallet not found", http.StatusNotFound)
			return
		}
		if !f.applied[req.Reference] {
			if f.declineTransfer || from.Balance < req.Amount {
				http.Error(w, "insufficient funds", http.StatusBadRequest)
				return
			}
			from.Balance -= req.Amount
			to.Balance += req.Amount
			f.applied[req.Reference] = true
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"fromWallet": f.walletJSON(from),
			"toWallet":   f.walletJSON(to),
		})
	})

	users := chi.NewRouter()
	users.Get("/api/users/auth/search/omnitag/{omnitag}", func(w http.ResponseWriter, r *http.Request) {
		tag := chi.URLParam(r, "omnitag")
		f.mu.Lock()
		accountID, ok := f.omniTags[tag]
		f.mu.Unlock()
		if !ok {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"accountId": accountID, "omniTag": tag})
	})

	fraud := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"transactionId": "test",
			"riskScore":     5.0,
			"riskLevel":     "low",
			"decision":      "approve",
		})
	})

	walletSrv := httptest.NewServer(wallets)
	usersSrv := httptest.NewServer(users)
	fraudSrv := httptest.NewServer(fraud)
	t.Cleanup(walletSrv.Close)
	t.Cleanup(usersSrv.Close)
	t.Cleanup(fraudSrv.Close)

	t.Setenv("WALLET_SERVICE_URL", walletSrv.URL)
	t.Setenv("USER_SERVICE_URL", usersSrv.URL)
	t.Setenv("FRAUD_DETECTION_URL", fraudSrv.URL)
}

func (f *fakeBackend) balance(walletID string) float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.wallets[walletID].Balance
}

func (f *fakeBackend) set(mutate func(*fakeBackend)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	mutate(f)
}

func setupTransfer(t *testing.T) *fakeBackend {
	t.Helper()
	if err := db.Init(); err != nil {
		t.Fatalf("db.Init: %v", err)
	}
	f := &fakeBackend{
		wallets: map[string]*fakeWallet{
			"w-sender":   {WalletID: "w-sender", AccountID: "acc-sender", Balance: 100, Status: "active", IsDefault: true},
			"w-receiver": {WalletID: "w-receiver", AccountID: "acc-receiver", Balance: 50, Status: "active", IsDefault: true},
		},
		omniTags: map[string]string{"jane": "acc-receiver"},
		applied:  map[string]bool{},
	}
	f.start(t)
	return f
}

func transferReq(key string, amount float64) models.TransferRequest {
	return models.TransferRequest{
		SenderWalletID:  "w-sender",
		ReceiverOmniTag: "jane",
		Amount:          amount,
		Description:     "test transfer",
		IdempotencyKey:  key,
	}
}

func doTransfer(t *testing.T, req models.TransferRequest) (*httptest.ResponseRecorder, *models.TransferResponse) {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal transfer request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/api/transactions/transfer", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handlers.HandlerTransferMoney(rec, httpReq)

	var resp models.TransferResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec, &resp
}

func senderTransactions(t *testing.T) []*models.Transaction {
	t.Helper()
	txs, err := db.GetTransactionsByAccountID("acc-sender", &models.TransactionHistoryParams{Limit: 10})
	if err != nil {
		t.Fatalf("GetTransactionsByAccountID: %v", err)
	}
	return txs
}

func TestTransferMovesMoney(t *testing.T) {
	f := setupTransfer(t)

	rec, resp := doTransfer(t, transferReq("key-happy", 40))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if resp.Status != "success" {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Message)
	}
	if resp.SenderBalance != 60 || resp.ReceiverBalance != 90 {
		t.Errorf("expected balances 60/90 in response, got %.2f/%.2f", resp.SenderBalance, resp.ReceiverBalance)
	}
	if got := f.balance("w-sender"); got != 60 {
		t.Errorf("sender wallet balance: expected 60, got %.2f", got)
	}
	if got := f.balance("w-receiver"); got != 90 {
		t.Errorf("receiver wallet balance: expected 90, got %.2f", got)
	}
	if total := f.balance("w-sender") + f.balance("w-receiver"); total != 150 {
		t.Errorf("money not conserved: total %.2f != 150", total)
	}

	tx, err := db.GetTransaction(resp.TransactionID)
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	if tx.Status != models.TransactionStatusCompleted {
		t.Errorf("expected completed transaction, got %s", tx.Status)
	}
	if tx.BalanceAfter != 60 {
		t.Errorf("expected BalanceAfter 60, got %.2f", tx.BalanceAfter)
	}
}

func TestTransferReceiverNotFound(t *testing.T) {
	f := setupTransfer(t)

	req := transferReq("key-ghost", 40)
	req.ReceiverOmniTag = "ghost"
	rec, resp := doTransfer(t, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if resp.Status != "failed" || resp.Message != "Receiver not found" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if len(f.transferCalls) != 0 {
		t.Errorf("no wallet transfer must happen, got %v", f.transferCalls)
	}
	if got := f.balance("w-sender"); got != 100 {
		t.Errorf("sender balance must be untouched, got %.2f", got)
	}
}

func TestTransferInsufficientBalance(t *testing.T) {
	f := setupTransfer(t)

	rec, resp := doTransfer(t, transferReq("key-poor", 500))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if resp.Message != "Insufficient balance" {
		t.Errorf("unexpected message: %s", resp.Message)
	}
	if len(f.transferCalls) != 0 {
		t.Errorf("no wallet transfer must happen, got %v", f.transferCalls)
	}
}

// The wallet service is the authority on funds: if it declines even though
// the earlier read looked fine, the transaction must be recorded failed and
// no balance may change (the wallet op is atomic — nothing to compensate).
func TestTransferWalletDeclines(t *testing.T) {
	f := setupTransfer(t)
	f.set(func(f *fakeBackend) { f.declineTransfer = true })

	rec, resp := doTransfer(t, transferReq("key-decline", 40))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if resp.Status != "failed" || !strings.Contains(resp.Message, "insufficient funds") {
		t.Errorf("unexpected response: %+v", resp)
	}
	if f.balance("w-sender") != 100 || f.balance("w-receiver") != 50 {
		t.Errorf("balances must be untouched, got %.2f/%.2f", f.balance("w-sender"), f.balance("w-receiver"))
	}

	txs := senderTransactions(t)
	if len(txs) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txs))
	}
	if txs[0].Status != models.TransactionStatusFailed {
		t.Errorf("expected failed transaction, got %s", txs[0].Status)
	}
	if txs[0].FailedReason == "" {
		t.Error("expected FailedReason to be set")
	}
}

// A wallet-service outage must return 502 and NOT consume the idempotency key,
// so the same request succeeds once the service is back.
func TestTransferRetriesAfterWalletOutage(t *testing.T) {
	f := setupTransfer(t)
	f.set(func(f *fakeBackend) { f.failTransfer = true })

	rec, resp := doTransfer(t, transferReq("key-retry", 40))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
	if resp.Status != "failed" {
		t.Errorf("expected failed, got %+v", resp)
	}
	if got := f.balance("w-sender"); got != 100 {
		t.Errorf("sender balance must be untouched, got %.2f", got)
	}

	f.set(func(f *fakeBackend) { f.failTransfer = false })

	rec, resp = doTransfer(t, transferReq("key-retry", 40))
	if rec.Code != http.StatusOK || resp.Status != "success" {
		t.Fatalf("retry with same key must succeed, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := f.balance("w-sender"); got != 60 {
		t.Errorf("expected sender balance 60 after retry, got %.2f", got)
	}
}

func TestTransferIdempotentReplay(t *testing.T) {
	f := setupTransfer(t)

	rec, _ := doTransfer(t, transferReq("key-dup", 40))
	if rec.Code != http.StatusOK {
		t.Fatalf("first transfer failed: %d %s", rec.Code, rec.Body.String())
	}

	rec, _ = doTransfer(t, transferReq("key-dup", 40))
	if rec.Code != http.StatusOK {
		t.Fatalf("replay must return cached response, got %d: %s", rec.Code, rec.Body.String())
	}

	if len(f.transferCalls) != 1 {
		t.Errorf("money must move exactly once, got %d wallet transfers", len(f.transferCalls))
	}
	if got := f.balance("w-sender"); got != 60 {
		t.Errorf("expected sender balance 60, got %.2f", got)
	}
}
