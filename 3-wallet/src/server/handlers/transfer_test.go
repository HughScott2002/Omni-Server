package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/m/v2/src/db"
	"example.com/m/v2/src/models"
	"example.com/m/v2/src/server/handlers"
	"github.com/go-chi/chi/v5"
)

func setupRouter(t *testing.T) http.Handler {
	t.Helper()
	t.Setenv("ENVIRONMENT", "local")
	t.Setenv("MODE", "memcached")
	if err := db.Init(); err != nil {
		t.Fatalf("db.Init: %v", err)
	}

	r := chi.NewRouter()
	r.Post("/api/wallets/transfer", handlers.HandlerWalletTransfer)
	return r
}

func seedWallet(t *testing.T, id string, balance float64, currency models.Currency, status models.WalletStatus) {
	t.Helper()
	err := db.AddWallet(&models.Wallet{
		WalletId:  id,
		AccountId: "acc-" + id,
		Type:      models.WalletTypePrimary,
		Balance:   balance,
		Currency:  currency,
		Status:    status,
		IsDefault: true,
	})
	if err != nil {
		t.Fatalf("seed wallet %s: %v", id, err)
	}
}

func seedPair(t *testing.T) {
	t.Helper()
	seedWallet(t, "w-from", 100, models.CurrencyUSD, models.WalletStatusActive)
	seedWallet(t, "w-to", 50, models.CurrencyUSD, models.WalletStatusActive)
}

func transfer(t *testing.T, router http.Handler, req models.WalletTransferRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/api/wallets/transfer", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httpReq)
	return rec
}

func validTransfer(reference string, amount float64) models.WalletTransferRequest {
	return models.WalletTransferRequest{
		FromWalletId: "w-from",
		ToWalletId:   "w-to",
		Amount:       amount,
		Reference:    reference,
	}
}

// assertBalances checks stored balances and that money was conserved.
func assertBalances(t *testing.T, fromBalance, toBalance float64) {
	t.Helper()
	from, err := db.GetWallet("w-from")
	if err != nil {
		t.Fatalf("GetWallet w-from: %v", err)
	}
	to, err := db.GetWallet("w-to")
	if err != nil {
		t.Fatalf("GetWallet w-to: %v", err)
	}
	if from.Balance != fromBalance || to.Balance != toBalance {
		t.Errorf("expected balances %.2f/%.2f, got %.2f/%.2f", fromBalance, toBalance, from.Balance, to.Balance)
	}
	if from.Balance+to.Balance != 150 {
		t.Errorf("money not conserved: %.2f + %.2f != 150", from.Balance, to.Balance)
	}
}

func TestTransferMovesMoneyAtomically(t *testing.T) {
	router := setupRouter(t)
	seedPair(t)

	rec := transfer(t, router, validTransfer("TXN-1", 40))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result models.WalletTransferResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.FromWallet.Balance != 60 || result.ToWallet.Balance != 90 {
		t.Errorf("expected 60/90 in response, got %.2f/%.2f", result.FromWallet.Balance, result.ToWallet.Balance)
	}
	assertBalances(t, 60, 90)
}

func TestTransferInsufficientFunds(t *testing.T) {
	router := setupRouter(t)
	seedPair(t)

	rec := transfer(t, router, validTransfer("TXN-1", 500))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "insufficient funds") {
		t.Errorf("expected insufficient funds message, got: %s", rec.Body.String())
	}
	assertBalances(t, 100, 50)
}

func TestTransferInactiveSender(t *testing.T) {
	router := setupRouter(t)
	seedWallet(t, "w-from", 100, models.CurrencyUSD, models.WalletStatusSuspended)
	seedWallet(t, "w-to", 50, models.CurrencyUSD, models.WalletStatusActive)

	rec := transfer(t, router, validTransfer("TXN-1", 40))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	assertBalances(t, 100, 50)
}

// An atomic transfer can safely refuse inactive receivers: there is no
// debited-but-not-credited state that would need a refund path.
func TestTransferInactiveReceiver(t *testing.T) {
	router := setupRouter(t)
	seedWallet(t, "w-from", 100, models.CurrencyUSD, models.WalletStatusActive)
	seedWallet(t, "w-to", 50, models.CurrencyUSD, models.WalletStatusSuspended)

	rec := transfer(t, router, validTransfer("TXN-1", 40))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	assertBalances(t, 100, 50)
}

func TestTransferCurrencyMismatch(t *testing.T) {
	router := setupRouter(t)
	seedWallet(t, "w-from", 100, models.CurrencyUSD, models.WalletStatusActive)
	seedWallet(t, "w-to", 50, models.CurrencyEUR, models.WalletStatusActive)

	rec := transfer(t, router, validTransfer("TXN-1", 40))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTransferUnknownWallet(t *testing.T) {
	router := setupRouter(t)
	seedWallet(t, "w-from", 100, models.CurrencyUSD, models.WalletStatusActive)

	rec := transfer(t, router, validTransfer("TXN-1", 40))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTransferSameWallet(t *testing.T) {
	router := setupRouter(t)
	seedPair(t)

	req := validTransfer("TXN-1", 40)
	req.ToWalletId = req.FromWalletId
	rec := transfer(t, router, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTransferValidation(t *testing.T) {
	router := setupRouter(t)
	seedPair(t)

	cases := []models.WalletTransferRequest{
		{FromWalletId: "", ToWalletId: "w-to", Amount: 40, Reference: "TXN-1"},
		{FromWalletId: "w-from", ToWalletId: "", Amount: 40, Reference: "TXN-1"},
		{FromWalletId: "w-from", ToWalletId: "w-to", Amount: 0, Reference: "TXN-1"},
		{FromWalletId: "w-from", ToWalletId: "w-to", Amount: -5, Reference: "TXN-1"},
		{FromWalletId: "w-from", ToWalletId: "w-to", Amount: 40, Reference: ""},
	}
	for i, req := range cases {
		if rec := transfer(t, router, req); rec.Code != http.StatusBadRequest {
			t.Errorf("case %d: expected 400, got %d", i, rec.Code)
		}
	}
	assertBalances(t, 100, 50)
}

// Replaying a reference must not move money twice — the original result comes
// back instead (client-assigned transfer identity, TigerBeetle-style).
func TestTransferIdempotentByReference(t *testing.T) {
	router := setupRouter(t)
	seedPair(t)

	first := transfer(t, router, validTransfer("TXN-DUP", 40))
	if first.Code != http.StatusOK {
		t.Fatalf("first transfer: %d %s", first.Code, first.Body.String())
	}

	replay := transfer(t, router, validTransfer("TXN-DUP", 40))
	if replay.Code != http.StatusOK {
		t.Fatalf("replay: %d %s", replay.Code, replay.Body.String())
	}

	var result models.WalletTransferResult
	if err := json.Unmarshal(replay.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode replay result: %v", err)
	}
	if result.FromWallet.Balance != 60 || result.ToWallet.Balance != 90 {
		t.Errorf("replay must return original result 60/90, got %.2f/%.2f", result.FromWallet.Balance, result.ToWallet.Balance)
	}
	assertBalances(t, 60, 90)
}
