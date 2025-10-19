package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"omni/src/db"
)

func TestHandlerRegister_Success(t *testing.T) {
	setupTestDB()

	// Create registration request
	registerReq := map[string]string{
		"email":     "newuser@example.com",
		"password":  "securepassword123",
		"firstName": "John",
		"lastName":  "Doe",
		"currency":  "USD",
	}
	reqBody, _ := json.Marshal(registerReq)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test)")
	w := httptest.NewRecorder()

	// Call handler
	HandlerRegister(w, req)

	// Verify response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
		t.Logf("Response: %s", w.Body.String())
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify user data in response
	user, ok := response["user"].(map[string]interface{})
	if !ok {
		t.Fatal("Response does not contain user data")
	}

	if user["email"] != "newuser@example.com" {
		t.Errorf("Expected email newuser@example.com, got %s", user["email"])
	}

	if user["firstName"] != "John" {
		t.Errorf("Expected firstName John, got %s", user["firstName"])
	}

	if user["kycStatus"] != "pending" {
		t.Errorf("Expected kycStatus pending, got %s", user["kycStatus"])
	}

	// Verify account ID was generated
	if user["id"] == "" {
		t.Error("Account ID was not generated")
	}

	// Verify session was created
	session, ok := response["session"].(map[string]interface{})
	if !ok {
		t.Fatal("Response does not contain session data")
	}

	if session["id"] == "" {
		t.Error("Session ID was not generated")
	}

	// Verify cookies are set (auto-login after registration)
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
		t.Error("Access token cookie not set after registration")
	}
	if !hasRefreshToken {
		t.Error("Refresh token cookie not set after registration")
	}
}

func TestHandlerRegister_DuplicateEmail(t *testing.T) {
	setupTestDB()

	// Create first user
	email := "duplicate@example.com"
	_, err := createTestUser(email, "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Attempt to register with same email
	registerReq := map[string]string{
		"email":     email,
		"password":  "differentpassword",
		"firstName": "Jane",
		"lastName":  "Doe",
	}
	reqBody, _ := json.Marshal(registerReq)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test)")
	w := httptest.NewRecorder()

	HandlerRegister(w, req)

	// Should return conflict error
	if w.Code != http.StatusConflict {
		t.Errorf("Expected status code %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestHandlerRegister_InvalidJSON(t *testing.T) {
	setupTestDB()

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerRegister(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandlerRegister_EmptyBody(t *testing.T) {
	setupTestDB()

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerRegister(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// BUG TEST: Registration should validate email format
func TestHandlerRegister_InvalidEmailFormat(t *testing.T) {
	setupTestDB()

	registerReq := map[string]string{
		"email":     "not-an-email",
		"password":  "password123",
		"firstName": "John",
		"lastName":  "Doe",
	}
	reqBody, _ := json.Marshal(registerReq)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test)")
	w := httptest.NewRecorder()

	HandlerRegister(w, req)

	// BUG: This test will FAIL because there's no email validation
	// Expected: Should return 400 Bad Request
	// Actual: Returns 201 Created
	if w.Code == http.StatusCreated {
		t.Error("BUG DETECTED: Invalid email format was accepted")
	}
}

// BUG TEST: Registration should enforce password strength
func TestHandlerRegister_WeakPassword(t *testing.T) {
	setupTestDB()

	registerReq := map[string]string{
		"email":     "weak@example.com",
		"password":  "123",
		"firstName": "John",
		"lastName":  "Doe",
	}
	reqBody, _ := json.Marshal(registerReq)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test)")
	w := httptest.NewRecorder()

	HandlerRegister(w, req)

	// BUG: This test will FAIL because there's no password strength validation
	// Expected: Should return 400 Bad Request
	// Actual: Returns 201 Created
	if w.Code == http.StatusCreated {
		t.Error("BUG DETECTED: Weak password was accepted")
	}
}

// Test password hashing
func TestHandlerRegister_PasswordIsHashed(t *testing.T) {
	setupTestDB()

	plainPassword := "mySecurePassword123"
	registerReq := map[string]string{
		"email":     "hash@example.com",
		"password":  plainPassword,
		"firstName": "John",
		"lastName":  "Doe",
	}
	reqBody, _ := json.Marshal(registerReq)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test)")
	w := httptest.NewRecorder()

	HandlerRegister(w, req)

	// Get the user from database
	user, err := db.GetUser("hash@example.com")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	// Verify password is hashed
	if user.HashedPassword == plainPassword {
		t.Error("Password was not hashed")
	}

	// Verify hashed password starts with bcrypt prefix
	if len(user.HashedPassword) < 20 {
		t.Error("Hashed password is too short to be a valid bcrypt hash")
	}
}

// Test account ID generation
func TestHandlerRegister_AccountIdGeneration(t *testing.T) {
	setupTestDB()

	registerReq := map[string]string{
		"email":     "accountid@example.com",
		"password":  "password123",
		"firstName": "John",
		"lastName":  "Doe",
	}
	reqBody, _ := json.Marshal(registerReq)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test)")
	w := httptest.NewRecorder()

	HandlerRegister(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	user := response["user"].(map[string]interface{})
	accountId := user["id"].(string)

	if accountId == "" {
		t.Error("Account ID was not generated")
	}

	// Account ID should be unique
	// Register another user
	registerReq2 := map[string]string{
		"email":     "accountid2@example.com",
		"password":  "password123",
		"firstName": "Jane",
		"lastName":  "Doe",
	}
	reqBody2, _ := json.Marshal(registerReq2)

	req2 := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(reqBody2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Test)")
	w2 := httptest.NewRecorder()

	HandlerRegister(w2, req2)

	var response2 map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &response2)

	user2 := response2["user"].(map[string]interface{})
	accountId2 := user2["id"].(string)

	if accountId == accountId2 {
		t.Error("Account IDs are not unique")
	}
}
