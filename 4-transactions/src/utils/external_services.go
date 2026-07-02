package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// UserInfo represents basic user information from the user service
type UserInfo struct {
	AccountID string `json:"accountId"`
	OmniTag   string `json:"omniTag"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
}

// Wallet represents basic wallet information from the wallet service
type Wallet struct {
	WalletID     string  `json:"walletId"`
	AccountID    string  `json:"accountId"`
	Balance      float64 `json:"balance"`
	Currency     string  `json:"currency"`
	Status       string  `json:"status"`
	IsDefault    bool    `json:"isDefault"`
	DailyLimit   float64 `json:"dailyLimit"`
	MonthlyLimit float64 `json:"monthlyLimit"`
}

// VirtualCard represents basic card information from the wallet service
type VirtualCard struct {
	ID           string  `json:"id"`
	WalletID     string  `json:"walletId"`
	CardType     string  `json:"cardType"`
	CardBrand    string  `json:"cardBrand"`
	Currency     string  `json:"currency"`
	CardStatus   string  `json:"cardStatus"`
	Balance      float64 `json:"balance"`
	DailyLimit   float64 `json:"dailyLimit"`
	MonthlyLimit float64 `json:"monthlyLimit"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// ErrWalletRejected marks business declines from the wallet service (4xx),
// as opposed to the service being unreachable.
var ErrWalletRejected = errors.New("wallet operation rejected")

// Service URLs are read per call so tests can point them at httptest servers.
func userServiceURL() string {
	return getEnv("USER_SERVICE_URL", "http://user-service:8080")
}

func walletServiceURL() string {
	return getEnv("WALLET_SERVICE_URL", "http://wallet-service:8080")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetUserByOmniTag fetches user information by OmniTag from the user service
func GetUserByOmniTag(omniTag string) (*UserInfo, error) {
	url := fmt.Sprintf("%s/api/users/auth/search/omnitag/%s", userServiceURL(), omniTag)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call user service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user service returned status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %v", err)
	}

	return &userInfo, nil
}

// GetWallet fetches wallet information by wallet ID from the wallet service
func GetWallet(walletID string) (*Wallet, error) {
	url := fmt.Sprintf("%s/api/wallets/%s", walletServiceURL(), walletID)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call wallet service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("wallet service returned status %d: %s", resp.StatusCode, string(body))
	}

	var wallet Wallet
	if err := json.NewDecoder(resp.Body).Decode(&wallet); err != nil {
		return nil, fmt.Errorf("failed to decode wallet info: %v", err)
	}

	return &wallet, nil
}

// GetDefaultWallet fetches the default wallet for an account
func GetDefaultWallet(accountID string) (*Wallet, error) {
	url := fmt.Sprintf("%s/api/wallets/list/%s", walletServiceURL(), accountID)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call wallet service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("wallet service returned status %d: %s", resp.StatusCode, string(body))
	}

	var wallets []Wallet
	if err := json.NewDecoder(resp.Body).Decode(&wallets); err != nil {
		return nil, fmt.Errorf("failed to decode wallets: %v", err)
	}

	// Find the default wallet
	for _, wallet := range wallets {
		if wallet.IsDefault {
			return &wallet, nil
		}
	}

	// If no default wallet found, return the first one
	if len(wallets) > 0 {
		return &wallets[0], nil
	}

	return nil, fmt.Errorf("no wallets found for account %s", accountID)
}

// GetVirtualCard fetches card information by card ID from the wallet service
func GetVirtualCard(cardID string) (*VirtualCard, error) {
	url := fmt.Sprintf("%s/api/wallets/cards/%s", walletServiceURL(), cardID)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call wallet service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("wallet service returned status %d: %s", resp.StatusCode, string(body))
	}

	var card VirtualCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("failed to decode card info: %v", err)
	}

	return &card, nil
}

// WalletTransferResult carries both wallets as returned by the wallet
// service after an atomic transfer.
type WalletTransferResult struct {
	FromWallet Wallet `json:"fromWallet"`
	ToWallet   Wallet `json:"toWallet"`
}

// TransferBetweenWallets moves money between two wallets in one atomic wallet
// service operation: both legs apply or neither does. The reference is the
// transfer's identity — replaying the same reference is a no-op that returns
// the original result, so this call is safe to retry.
func TransferBetweenWallets(fromWalletID, toWalletID string, amount float64, reference string) (*WalletTransferResult, error) {
	url := fmt.Sprintf("%s/api/wallets/transfer", walletServiceURL())

	body, err := json.Marshal(map[string]interface{}{
		"fromWalletId": fromWalletID,
		"toWalletId":   toWalletID,
		"amount":       amount,
		"reference":    reference,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transfer request: %v", err)
	}

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to call wallet service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %s", ErrWalletRejected, strings.TrimSpace(string(msg)))
	}

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("wallet service returned status %d: %s", resp.StatusCode, string(msg))
	}

	var result WalletTransferResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode transfer result: %v", err)
	}

	return &result, nil
}
