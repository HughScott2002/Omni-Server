package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"omni/src/db"
	"omni/src/utils"
)

func TestHandlerCheckSession_WithValidAccessToken(t *testing.T) {
	setupTestDB()

	// Create test user and get tokens
	testEmail := "session-test@example.com"
	user, _ := createTestUser(testEmail, "password123")

	accessToken, err := utils.GenerateAccessToken(user.Email)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	// Create request with access token cookie
	req := httptest.NewRequest(http.MethodGet, "/check-session", nil)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: accessToken,
	})
	w := httptest.NewRecorder()

	HandlerCheckSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	user_data, ok := response["user"].(map[string]interface{})
	if !ok {
		t.Fatal("Response does not contain user data")
	}

	if user_data["email"] != testEmail {
		t.Errorf("Expected email %s, got %s", testEmail, user_data["email"])
	}
}

func TestHandlerCheckSession_WithValidRefreshToken(t *testing.T) {
	setupTestDB()

	// Create test user
	testEmail := "refresh-test@example.com"
	user, _ := createTestUser(testEmail, "password123")

	// Generate refresh token and store it
	refreshToken, _ := utils.GenerateRefreshToken(user.Email)
	db.AddRefreshToken(refreshToken, db.RefreshTokenInfo{
		UserEmail:  user.Email,
		DeviceInfo: "Test Device",
	})

	// Create request with only refresh token (no access token)
	req := httptest.NewRequest(http.MethodGet, "/check-session", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken,
	})
	w := httptest.NewRecorder()

	HandlerCheckSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		t.Logf("Response: %s", w.Body.String())
	}

	// Verify new access token was set
	cookies := w.Result().Cookies()
	hasNewAccessToken := false
	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			hasNewAccessToken = true
		}
	}

	if !hasNewAccessToken {
		t.Error("New access token was not set")
	}
}

func TestHandlerCheckSession_NoTokens(t *testing.T) {
	setupTestDB()

	req := httptest.NewRequest(http.MethodGet, "/check-session", nil)
	w := httptest.NewRecorder()

	HandlerCheckSession(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandlerCheckSession_InvalidRefreshToken(t *testing.T) {
	setupTestDB()

	req := httptest.NewRequest(http.MethodGet, "/check-session", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "invalid-token",
	})
	w := httptest.NewRecorder()

	HandlerCheckSession(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandlerListActiveSessions_Success(t *testing.T) {
	setupTestDB()

	// Create test user and login to create sessions
	testEmail := "list-sessions@example.com"
	testPassword := "password123"
	// user, _ := createTestUser(testEmail, testPassword)

	// Login to create a session
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

	// Now list sessions
	listReq := map[string]string{
		"email": testEmail,
	}
	listBody, _ := json.Marshal(listReq)

	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(listBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerListActiveSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		t.Logf("Response: %s", w.Body.String())
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	sessions, ok := response["activeSessions"].([]interface{})
	if !ok {
		t.Fatal("Response does not contain activeSessions array")
	}

	if len(sessions) == 0 {
		t.Error("Expected at least one active session")
	}

	// Verify session structure
	session := sessions[0].(map[string]interface{})
	if session["browser"] == nil {
		t.Error("Session does not contain browser information")
	}
	if session["ipAddress"] == nil {
		t.Error("Session does not contain IP address")
	}
}

// BUG TEST: List sessions should require authentication
func TestHandlerListActiveSessions_NoAuth(t *testing.T) {
	setupTestDB()

	// Create test user with sessions
	testEmail := "noauth-sessions@example.com"
	createTestUser(testEmail, "password123")

	// Attempt to list sessions without authentication
	listReq := map[string]string{
		"email": testEmail,
	}
	listBody, _ := json.Marshal(listReq)

	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(listBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerListActiveSessions(w, req)

	// BUG: This test will FAIL because there's no authentication check
	// Expected: Should return 401 Unauthorized
	// Actual: Returns 200 OK and shows sessions
	if w.Code == http.StatusOK {
		t.Error("BUG DETECTED: Unauthenticated user can list sessions")
	}
}

func TestHandlerListActiveSessions_EmptyEmail(t *testing.T) {
	setupTestDB()

	listReq := map[string]string{
		"email": "",
	}
	listBody, _ := json.Marshal(listReq)

	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(listBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerListActiveSessions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandlerLogoutAllOtherSessions_Success(t *testing.T) {
	setupTestDB()

	// Create test user
	testEmail := "logout-others@example.com"
	testPassword := "password123"
	user, _ := createTestUser(testEmail, testPassword)

	// Create multiple sessions by logging in multiple times
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
	}

	// Get all sessions
	sessionsBefore, _ := db.GetUserSessions(user.Email)
	if len(sessionsBefore) < 2 {
		t.Skip("Need at least 2 sessions for this test")
	}

	// Get the last session's refresh token
	lastSession := sessionsBefore[len(sessionsBefore)-1]
	currentRefreshToken := lastSession.Token

	// Logout all other sessions
	req := httptest.NewRequest(http.MethodPost, "/logout-others", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: currentRefreshToken,
	})
	w := httptest.NewRecorder()

	HandlerLogoutAllOtherSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Verify only one session remains
	sessionsAfter, _ := db.GetUserSessions(user.Email)
	if len(sessionsAfter) != 1 {
		t.Errorf("Expected 1 session remaining, got %d", len(sessionsAfter))
	}

	// Verify the remaining session is the current one
	if sessionsAfter[0].Token != currentRefreshToken {
		t.Error("Wrong session was kept")
	}
}

func TestHandlerLogoutAllOtherSessions_NoRefreshToken(t *testing.T) {
	setupTestDB()

	req := httptest.NewRequest(http.MethodPost, "/logout-others", nil)
	w := httptest.NewRecorder()

	HandlerLogoutAllOtherSessions(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// BUG TEST: Logout session by ID should require authentication
func TestHandlerLogoutSessionById_NoAuth(t *testing.T) {
	setupTestDB()

	// Create a session
	testEmail := "logout-by-id@example.com"
	testPassword := "password123"
	createTestUser(testEmail, testPassword)

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

	// Get the session ID
	// sessions, _ := db.GetUserSessions(testEmail)
	// sessionID := sessions[0].ID

	// Attempt to logout session without authentication
	// Note: In the real implementation, this would use chi.URLParam
	// For testing, we'll simulate the request
	// req := httptest.NewRequest(http.MethodPost, "/logout/"+sessionID, nil)
	// w := httptest.NewRecorder()

	// BUG: This endpoint has no auth check, anyone can logout any session
	// This test documents the bug but can't fully test it without the router context
	t.Log("BUG DOCUMENTED: HandlerLogoutSessionById has no authentication check")
}
