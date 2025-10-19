package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

var (
	userServiceURL   = getEnv("USER_SERVICE_URL", "http://users:8080")
	walletServiceURL = getEnv("WALLET_SERVICE_URL", "http://wallets:8082")
	httpClient       = &http.Client{Timeout: 10 * time.Second}
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetUserByOmniTag fetches user information by OmniTag from the user service
func GetUserByOmniTag(omniTag string) (*UserInfo, error) {
	url := fmt.Sprintf("%s/api/users/search/%s", userServiceURL, omniTag)

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
	url := fmt.Sprintf("%s/api/wallets/%s", walletServiceURL, walletID)

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
	url := fmt.Sprintf("%s/api/wallets/list/%s", walletServiceURL, accountID)

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
	url := fmt.Sprintf("%s/api/wallets/cards/%s", walletServiceURL, cardID)

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

// UpdateWalletBalance updates the wallet balance via the wallet service
// Note: This is a simplified version. In production, you'd want proper wallet update endpoints
func UpdateWalletBalance(walletID string, newBalance float64) error {
	// In a real implementation, you'd call a proper wallet update endpoint
	// For now, we'll assume the wallet service has an internal update mechanism
	// or we'd need to add a specific endpoint for balance updates

	// This is a placeholder - you'll need to implement the actual API call
	// based on how the wallet service exposes balance updates
	return nil
}
