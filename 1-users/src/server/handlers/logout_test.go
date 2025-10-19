package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"omni/src/db"
)

func TestHandlerLogout_Success(t *testing.T) {
	setupTestDB()

	// Create user and login
	testEmail := "logout@example.com"
	testPassword := "password123"
	createTestUser(testEmail, testPassword)

	// Login to create session and tokens
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

	// Get the refresh token from login
	var refreshToken string
	for _, cookie := range loginW.Result().Cookies() {
		if cookie.Name == "refresh_token" {
			refreshToken = cookie.Value
			break
		}
	}

	// Verify session exists
	sessionsBefore, _ := db.GetUserSessions(testEmail)
	if len(sessionsBefore) == 0 {
		t.Fatal("No session was created during login")
	}

	// Verify refresh token exists
	_, err := db.GetRefreshToken(refreshToken)
	if err != nil {
		t.Fatal("Refresh token was not stored")
	}

	// Logout
	logoutReq := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logoutReq.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken,
	})
	logoutW := httptest.NewRecorder()

	HandlerLogout(logoutW, logoutReq)

	// Verify response
	if logoutW.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, logoutW.Code)
	}

	var response map[string]string
	err = json.Unmarshal(logoutW.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["message"] != "Logged out successfully" {
		t.Errorf("Expected message 'Logged out successfully', got %s", response["message"])
	}

	// Verify session was deleted
	sessionsAfter, _ := db.GetUserSessions(testEmail)
	if len(sessionsAfter) != 0 {
		t.Errorf("Expected 0 sessions after logout, got %d", len(sessionsAfter))
	}

	// Verify refresh token was deleted
	_, err = db.GetRefreshToken(refreshToken)
	if err == nil {
		t.Error("Refresh token was not deleted")
	}

	// Verify cookies are cleared
	cookies := logoutW.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "access_token" || cookie.Name == "refresh_token" {
			if cookie.MaxAge != -1 {
				t.Errorf("Cookie %s was not cleared (MaxAge should be -1, got %d)", cookie.Name, cookie.MaxAge)
			}
			if cookie.Value != "" {
				t.Errorf("Cookie %s value should be empty, got %s", cookie.Name, cookie.Value)
			}
		}
	}
}

func TestHandlerLogout_NoRefreshToken(t *testing.T) {
	setupTestDB()

	// Logout without refresh token cookie
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	w := httptest.NewRecorder()

	HandlerLogout(w, req)

	// Should still succeed (graceful handling)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d even without cookie, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "Logged out successfully" {
		t.Errorf("Expected success message, got %s", response["message"])
	}
}

func TestHandlerLogout_InvalidRefreshToken(t *testing.T) {
	setupTestDB()

	// Logout with invalid refresh token
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "invalid-token-xyz",
	})
	w := httptest.NewRecorder()

	HandlerLogout(w, req)

	// Should still succeed (graceful handling)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d even with invalid token, got %d", http.StatusOK, w.Code)
	}
}

func TestHandlerLogout_MultipleSessions(t *testing.T) {
	setupTestDB()

	// Create user
	testEmail := "multi-logout@example.com"
	testPassword := "password123"
	createTestUser(testEmail, testPassword)

	// Login multiple times to create multiple sessions
	var refreshTokens []string
	for i := 0; i < 3; i++ {
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

		for _, cookie := range loginW.Result().Cookies() {
			if cookie.Name == "refresh_token" {
				refreshTokens = append(refreshTokens, cookie.Value)
				break
			}
		}
	}

	// Verify multiple sessions exist
	sessionsBefore, _ := db.GetUserSessions(testEmail)
	if len(sessionsBefore) < 2 {
		t.Skip("Need at least 2 sessions for this test")
	}

	// Logout from one session
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshTokens[0],
	})
	w := httptest.NewRecorder()

	HandlerLogout(w, req)

	// Verify only one session was deleted
	sessionsAfter, _ := db.GetUserSessions(testEmail)
	expectedCount := len(sessionsBefore) - 1
	if len(sessionsAfter) != expectedCount {
		t.Errorf("Expected %d sessions after logout, got %d", expectedCount, len(sessionsAfter))
	}

	// Verify the specific refresh token was deleted
	_, err := db.GetRefreshToken(refreshTokens[0])
	if err == nil {
		t.Error("Logged out refresh token still exists")
	}

	// Verify other refresh tokens still exist
	for i := 1; i < len(refreshTokens); i++ {
		_, err := db.GetRefreshToken(refreshTokens[i])
		if err != nil {
			t.Errorf("Other refresh token %d was incorrectly deleted", i)
		}
	}
}

func TestHandlerLogout_CookieClearing(t *testing.T) {
	setupTestDB()

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	w := httptest.NewRecorder()

	HandlerLogout(w, req)

	// Verify both cookies are cleared
	cookies := w.Result().Cookies()

	hasAccessTokenClear := false
	hasRefreshTokenClear := false

	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			hasAccessTokenClear = true
			if cookie.MaxAge != -1 {
				t.Error("access_token cookie MaxAge should be -1")
			}
		}
		if cookie.Name == "refresh_token" {
			hasRefreshTokenClear = true
			if cookie.MaxAge != -1 {
				t.Error("refresh_token cookie MaxAge should be -1")
			}
		}
	}

	if !hasAccessTokenClear {
		t.Error("access_token cookie was not cleared")
	}
	if !hasRefreshTokenClear {
		t.Error("refresh_token cookie was not cleared")
	}
}
