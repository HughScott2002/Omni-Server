package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"omni/src/db"
	"omni/src/models"
	"omni/src/utils"
)

func TestHandlerRefreshToken_Success(t *testing.T) {
	setupTestDB()

	// Create test user
	testEmail := "refresh@example.com"
	user, _ := createTestUser(testEmail, "password123")

	// Generate and store refresh token
	refreshToken, _ := utils.GenerateRefreshToken(user.Email)
	db.AddRefreshToken(refreshToken, db.RefreshTokenInfo{
		UserEmail:  user.Email,
		DeviceInfo: "Test Device",
	})

	// Create request with refresh token cookie
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken,
	})
	w := httptest.NewRecorder()

	HandlerRefreshToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		t.Logf("Response: %s", w.Body.String())
	}

	// Verify new cookies are set
	cookies := w.Result().Cookies()
	hasNewAccessToken := false
	hasNewRefreshToken := false

	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			hasNewAccessToken = true
		}
		if cookie.Name == "refresh_token" {
			hasNewRefreshToken = true
			// Verify it's a different token (token rotation)
			if cookie.Value == refreshToken {
				t.Error("Refresh token was not rotated")
			}
		}
	}

	if !hasNewAccessToken {
		t.Error("New access token cookie not set")
	}
	if !hasNewRefreshToken {
		t.Error("New refresh token cookie not set")
	}

	// Verify old refresh token was deleted
	_, err := db.GetRefreshToken(refreshToken)
	if err == nil {
		t.Error("Old refresh token was not deleted")
	}
}

func TestHandlerRefreshToken_NoRefreshToken(t *testing.T) {
	setupTestDB()

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	w := httptest.NewRecorder()

	HandlerRefreshToken(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandlerRefreshToken_InvalidRefreshToken(t *testing.T) {
	setupTestDB()

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "invalid-token-xyz",
	})
	w := httptest.NewRecorder()

	HandlerRefreshToken(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandlerRefreshToken_UserNotFound(t *testing.T) {
	setupTestDB()

	// Create refresh token for non-existent user
	fakeEmail := "nonexistent@example.com"
	refreshToken, _ := utils.GenerateRefreshToken(fakeEmail)
	db.AddRefreshToken(refreshToken, db.RefreshTokenInfo{
		UserEmail:  fakeEmail,
		DeviceInfo: "Test Device",
	})

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken,
	})
	w := httptest.NewRecorder()

	HandlerRefreshToken(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// BUG TEST: Session token should be updated when refresh token rotates
func TestHandlerRefreshToken_SessionTokenNotUpdated(t *testing.T) {
	setupTestDB()

	// Create user and session
	testEmail := "session-token@example.com"
	testPassword := "password123"
	user, _ := createTestUser(testEmail, testPassword)

	// Login to create session
	loginReq := map[string]string{
		"email":    testEmail,
		"password": testPassword,
	}
	reqBody, _ := json.Marshal(loginReq)
	loginRequest := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(reqBody))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRequest.Header.Set("User-Agent", "Mozilla/5.0 (Test)")
	loginW := httptest.NewRecorder()
	HandlerLogin(loginW, loginRequest)

	// Get the session
	sessionsBefore, _ := db.GetUserSessions(user.Email)
	if len(sessionsBefore) == 0 {
		t.Fatal("No session created")
	}
	oldSessionToken := sessionsBefore[0].Token

	// Get the refresh token from login response
	var oldRefreshToken string
	for _, cookie := range loginW.Result().Cookies() {
		if cookie.Name == "refresh_token" {
			oldRefreshToken = cookie.Value
			break
		}
	}

	// Refresh the token
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: oldRefreshToken,
	})
	w := httptest.NewRecorder()

	HandlerRefreshToken(w, req)

	// Get the new refresh token
	var newRefreshToken string
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "refresh_token" {
			newRefreshToken = cookie.Value
			break
		}
	}

	// Check if session token was updated
	sessionsAfter, _ := db.GetUserSessions(user.Email)
	if len(sessionsAfter) == 0 {
		t.Fatal("Session was deleted")
	}

	// BUG: Session token is not updated when refresh token rotates
	if sessionsAfter[0].Token == oldSessionToken {
		t.Error("BUG DETECTED: Session token was not updated after refresh token rotation")
		t.Logf("Old session token: %s", oldSessionToken)
		t.Logf("New session token: %s", sessionsAfter[0].Token)
		t.Logf("Old refresh token: %s", oldRefreshToken)
		t.Logf("New refresh token: %s", newRefreshToken)
	}
}

// BUG TEST: Should not allow refresh for disabled accounts
func TestHandlerRefreshToken_DisabledAccount(t *testing.T) {
	setupTestDB()

	// Create user
	testEmail := "disabled-refresh@example.com"
	user, _ := createTestUser(testEmail, "password123")

	// Generate refresh token
	refreshToken, _ := utils.GenerateRefreshToken(user.Email)
	db.AddRefreshToken(refreshToken, db.RefreshTokenInfo{
		UserEmail:  user.Email,
		DeviceInfo: "Test Device",
	})

	// Disable the account
	user.Status = models.AccountStatusDisabled
	db.UpdateUser(user)

	// Attempt to refresh
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken,
	})
	w := httptest.NewRecorder()

	HandlerRefreshToken(w, req)

	// BUG: Disabled accounts can still refresh tokens
	if w.Code == http.StatusOK {
		t.Error("BUG DETECTED: Disabled account was allowed to refresh token")
	}
}

// BUG TEST: Should not allow refresh for pending deletion accounts
func TestHandlerRefreshToken_PendingDeletionAccount(t *testing.T) {
	setupTestDB()

	// Create user
	testEmail := "pending-refresh@example.com"
	user, _ := createTestUser(testEmail, "password123")

	// Generate refresh token
	refreshToken, _ := utils.GenerateRefreshToken(user.Email)
	db.AddRefreshToken(refreshToken, db.RefreshTokenInfo{
		UserEmail:  user.Email,
		DeviceInfo: "Test Device",
	})

	// Mark for deletion
	user.RequestDeletion()
	db.UpdateUser(user)

	// Attempt to refresh
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken,
	})
	w := httptest.NewRecorder()

	HandlerRefreshToken(w, req)

	// BUG: Pending deletion accounts can still refresh tokens
	if w.Code == http.StatusOK {
		t.Error("BUG DETECTED: Pending deletion account was allowed to refresh token")
	}
}

func TestHandlerRefreshToken_TokenRotation(t *testing.T) {
	setupTestDB()

	// Create user
	testEmail := "rotation@example.com"
	user, _ := createTestUser(testEmail, "password123")

	// Generate initial refresh token
	refreshToken1, _ := utils.GenerateRefreshToken(user.Email)
	db.AddRefreshToken(refreshToken1, db.RefreshTokenInfo{
		UserEmail:  user.Email,
		DeviceInfo: "Test Device",
	})

	// First refresh
	req1 := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req1.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken1,
	})
	w1 := httptest.NewRecorder()

	HandlerRefreshToken(w1, req1)

	// Get new refresh token
	var refreshToken2 string
	for _, cookie := range w1.Result().Cookies() {
		if cookie.Name == "refresh_token" {
			refreshToken2 = cookie.Value
			break
		}
	}

	// Attempt to use old refresh token again
	req2 := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req2.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken1,
	})
	w2 := httptest.NewRecorder()

	HandlerRefreshToken(w2, req2)

	// Old token should not work
	if w2.Code != http.StatusUnauthorized {
		t.Error("Old refresh token should not work after rotation")
	}

	// New token should work
	req3 := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req3.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken2,
	})
	w3 := httptest.NewRecorder()

	HandlerRefreshToken(w3, req3)

	if w3.Code != http.StatusOK {
		t.Error("New refresh token should work")
	}
}
