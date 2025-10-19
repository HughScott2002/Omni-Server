package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"omni/src/db"
	"omni/src/models"
	"golang.org/x/crypto/bcrypt"
)

func setupTestDB() {
	// Initialize in-memory database for testing
	db.Init()
}

func createTestUser(email, password string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		AccountId:      "TEST123456",
		Email:          email,
		FirstName:      "Test",
		LastName:       "User",
		HashedPassword: string(hashedPassword),
		KYCStatus:      models.KYCStatusPending,
		Status:         models.AccountStatusActive,
	}

	err = db.AddUser(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func TestHandlerLogin_Success(t *testing.T) {
	setupTestDB()

	// Create test user
	testEmail := "test@example.com"
	testPassword := "password123"
	_, err := createTestUser(testEmail, testPassword)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create login request
	loginReq := map[string]string{
		"email":    testEmail,
		"password": testPassword,
	}
	reqBody, _ := json.Marshal(loginReq)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test)")
	w := httptest.NewRecorder()

	// Call handler
	HandlerLogin(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify user data in response
	user, ok := response["user"].(map[string]interface{})
	if !ok {
		t.Fatal("Response does not contain user data")
	}

	if user["email"] != testEmail {
		t.Errorf("Expected email %s, got %s", testEmail, user["email"])
	}

	// Verify cookies are set
	cookies := w.Result().Cookies()
	hasAccessToken := false
	hasRefreshToken := false

	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			hasAccessToken = true
		}
		if cookie.Name == "refresh_token" {
			hasRefreshToken = true
		}
	}

	if !hasAccessToken {
		t.Error("Access token cookie not set")
	}
	if !hasRefreshToken {
		t.Error("Refresh token cookie not set")
	}
}

func TestHandlerLogin_InvalidPassword(t *testing.T) {
	setupTestDB()

	// Create test user
	testEmail := "test2@example.com"
	testPassword := "password123"
	_, err := createTestUser(testEmail, testPassword)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create login request with wrong password
	loginReq := map[string]string{
		"email":    testEmail,
		"password": "wrongpassword",
	}
	reqBody, _ := json.Marshal(loginReq)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Call handler
	HandlerLogin(w, req)

	// Verify response
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandlerLogin_UserNotFound(t *testing.T) {
	setupTestDB()

	// Create login request for non-existent user
	loginReq := map[string]string{
		"email":    "nonexistent@example.com",
		"password": "password123",
	}
	reqBody, _ := json.Marshal(loginReq)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Call handler
	HandlerLogin(w, req)

	// Verify response
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandlerLogin_InvalidJSON(t *testing.T) {
	setupTestDB()

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerLogin(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandlerLogin_EmptyBody(t *testing.T) {
	setupTestDB()

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerLogin(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// BUG TEST: Login should check account status
func TestHandlerLogin_DisabledAccount(t *testing.T) {
	setupTestDB()

	// Create test user with disabled status
	testEmail := "disabled@example.com"
	testPassword := "password123"
	user, err := createTestUser(testEmail, testPassword)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Disable the account
	user.Status = models.AccountStatusDisabled
	db.UpdateUser(user)

	// Attempt login
	loginReq := map[string]string{
		"email":    testEmail,
		"password": testPassword,
	}
	reqBody, _ := json.Marshal(loginReq)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerLogin(w, req)

	// BUG: This test will FAIL because login doesn't check account status
	// Expected behavior: Should return 403 Forbidden
	// Actual behavior: Returns 200 OK and allows login
	if w.Code == http.StatusOK {
		t.Error("BUG DETECTED: Disabled account was allowed to login")
	}
}

// BUG TEST: Login should check for pending deletion accounts
func TestHandlerLogin_PendingDeletionAccount(t *testing.T) {
	setupTestDB()

	// Create test user
	testEmail := "pending-delete@example.com"
	testPassword := "password123"
	user, err := createTestUser(testEmail, testPassword)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Mark account for deletion
	user.RequestDeletion()
	db.UpdateUser(user)

	// Attempt login
	loginReq := map[string]string{
		"email":    testEmail,
		"password": testPassword,
	}
	reqBody, _ := json.Marshal(loginReq)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerLogin(w, req)

	// BUG: This test will FAIL because login doesn't check account status
	if w.Code == http.StatusOK {
		t.Error("BUG DETECTED: Account pending deletion was allowed to login")
	}
}

// Test session creation
func TestHandlerLogin_SessionCreation(t *testing.T) {
	setupTestDB()

	// Create test user
	testEmail := "session@example.com"
	testPassword := "password123"
	user, err := createTestUser(testEmail, testPassword)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Perform login
	loginReq := map[string]string{
		"email":    testEmail,
		"password": testPassword,
	}
	reqBody, _ := json.Marshal(loginReq)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	HandlerLogin(w, req)

	// Verify session was created
	sessions, err := db.GetUserSessions(user.Email)
	if err != nil {
		t.Fatalf("Failed to get user sessions: %v", err)
	}

	if len(sessions) == 0 {
		t.Error("No session was created after login")
	}

	// Verify session contains correct information
	session := sessions[0]
	if session.UserEmail != testEmail {
		t.Errorf("Expected session email %s, got %s", testEmail, session.UserEmail)
	}

	if session.DeviceInfo == "" {
		t.Error("Session device info is empty")
	}
}
