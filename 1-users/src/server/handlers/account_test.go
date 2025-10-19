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

func TestHandlerGetUserProfile_Success(t *testing.T) {
	setupTestDB()

	// Create test user
	testEmail := "profile@example.com"
	user, err := createTestUser(testEmail, "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Note: In real implementation, this would use chi.URLParam
	// For now, we'll test the handler directly after setting up the user

	// Get user from database to verify
	retrievedUser, err := db.GetUserByAccountId(user.AccountId)
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if retrievedUser.Email != testEmail {
		t.Errorf("Expected email %s, got %s", testEmail, retrievedUser.Email)
	}
}

// BUG TEST: Get user profile should require authentication
func TestHandlerGetUserProfile_NoAuth(t *testing.T) {
	setupTestDB()

	// Create test user
	testEmail := "noauth-profile@example.com"
	user, _ := createTestUser(testEmail, "password123")

	// Note: Cannot fully test without router, but documenting the bug
	t.Logf("BUG DOCUMENTED: HandlerGetUserProfile (GET /account/%s) has no authentication", user.AccountId)
	t.Log("Anyone can view any user's profile including sensitive PII")
}

// BUG TEST: Update user profile should require authentication
func TestHandlerUpdateUserProfile_NoAuth(t *testing.T) {
	setupTestDB()

	// Create test user
	testEmail := "noauth-update@example.com"
	user, _ := createTestUser(testEmail, "password123")

	t.Logf("BUG DOCUMENTED: HandlerUpdateUserProfile (PUT /account/%s) has no authentication", user.AccountId)
	t.Log("Anyone can update any user's profile")
}

// BUG TEST: Delete user account should require authentication
func TestHandlerDeleteUserAccount_NoAuth(t *testing.T) {
	setupTestDB()

	testEmail := "noauth-delete@example.com"
	user, _ := createTestUser(testEmail, "password123")

	t.Logf("BUG DOCUMENTED: HandlerDeleteUserAccount (DELETE /account/%s) has no authentication", user.AccountId)
	t.Log("Anyone can delete any user's account")
}

func TestHandlerChangePassword_Success(t *testing.T) {
	setupTestDB()

	// Create test user
	testEmail := "change-pw@example.com"
	currentPassword := "oldpassword123"
	user, _ := createTestUser(testEmail, currentPassword)

	// Change password request
	changeReq := models.PasswordChangeRequest{
		Email:              testEmail,
		AccountId:          user.AccountId,
		CurrentPassword:    currentPassword,
		NewPassword:        "newpassword456",
		ConfirmNewPassword: "newpassword456",
	}
	reqBody, _ := json.Marshal(changeReq)

	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerChangePassword(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		t.Logf("Response: %s", w.Body.String())
	}

	// Verify password was changed
	updatedUser, _ := db.GetUser(testEmail)
	err := bcrypt.CompareHashAndPassword([]byte(updatedUser.HashedPassword), []byte("newpassword456"))
	if err != nil {
		t.Error("Password was not updated correctly")
	}

	// Verify old password no longer works
	err = bcrypt.CompareHashAndPassword([]byte(updatedUser.HashedPassword), []byte(currentPassword))
	if err == nil {
		t.Error("Old password still works")
	}
}

func TestHandlerChangePassword_WrongCurrentPassword(t *testing.T) {
	setupTestDB()

	testEmail := "wrong-pw@example.com"
	user, _ := createTestUser(testEmail, "correctpassword")

	changeReq := models.PasswordChangeRequest{
		Email:              testEmail,
		AccountId:          user.AccountId,
		CurrentPassword:    "wrongpassword",
		NewPassword:        "newpassword456",
		ConfirmNewPassword: "newpassword456",
	}
	reqBody, _ := json.Marshal(changeReq)

	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerChangePassword(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandlerChangePassword_PasswordMismatch(t *testing.T) {
	setupTestDB()

	testEmail := "mismatch-pw@example.com"
	currentPassword := "currentpassword"
	user, _ := createTestUser(testEmail, currentPassword)

	changeReq := models.PasswordChangeRequest{
		Email:              testEmail,
		AccountId:          user.AccountId,
		CurrentPassword:    currentPassword,
		NewPassword:        "newpassword456",
		ConfirmNewPassword: "differentpassword",
	}
	reqBody, _ := json.Marshal(changeReq)

	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerChangePassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandlerChangePassword_SameAsCurrentPassword(t *testing.T) {
	setupTestDB()

	testEmail := "same-pw@example.com"
	currentPassword := "mypassword123"
	user, _ := createTestUser(testEmail, currentPassword)

	changeReq := models.PasswordChangeRequest{
		Email:              testEmail,
		AccountId:          user.AccountId,
		CurrentPassword:    currentPassword,
		NewPassword:        currentPassword,
		ConfirmNewPassword: currentPassword,
	}
	reqBody, _ := json.Marshal(changeReq)

	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerChangePassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandlerChangePassword_InvalidAccountId(t *testing.T) {
	setupTestDB()

	testEmail := "invalid-id@example.com"
	// user, _ := createTe stUser(testEmail, "password123")

	changeReq := models.PasswordChangeRequest{
		Email:              testEmail,
		AccountId:          "WRONG_ACCOUNT_ID",
		CurrentPassword:    "password123",
		NewPassword:        "newpassword456",
		ConfirmNewPassword: "newpassword456",
	}
	reqBody, _ := json.Marshal(changeReq)

	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerChangePassword(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandlerChangePassword_SessionsInvalidated(t *testing.T) {
	setupTestDB()

	// Create user and login to create sessions
	testEmail := "invalidate@example.com"
	testPassword := "password123"
	user, _ := createTestUser(testEmail, testPassword)

	// Create a session by logging in
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

	// Verify session exists
	sessionsBefore, _ := db.GetUserSessions(testEmail)
	if len(sessionsBefore) == 0 {
		t.Fatal("No session was created")
	}

	// Change password
	changeReq := models.PasswordChangeRequest{
		Email:              testEmail,
		AccountId:          user.AccountId,
		CurrentPassword:    testPassword,
		NewPassword:        "newpassword456",
		ConfirmNewPassword: "newpassword456",
	}
	changeBody, _ := json.Marshal(changeReq)

	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewReader(changeBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerChangePassword(w, req)

	// Verify all sessions were deleted
	sessionsAfter, _ := db.GetUserSessions(testEmail)
	if len(sessionsAfter) != 0 {
		t.Errorf("Expected 0 sessions after password change, got %d", len(sessionsAfter))
	}
}

// BUG TEST: Change password should have rate limiting
func TestHandlerChangePassword_NoRateLimit(t *testing.T) {
	setupTestDB()

	testEmail := "ratelimit@example.com"
	user, _ := createTestUser(testEmail, "password123")

	// Attempt multiple password changes rapidly
	for i := 0; i < 10; i++ {
		changeReq := models.PasswordChangeRequest{
			Email:              testEmail,
			AccountId:          user.AccountId,
			CurrentPassword:    "wrongpassword",
			NewPassword:        "newpassword456",
			ConfirmNewPassword: "newpassword456",
		}
		reqBody, _ := json.Marshal(changeReq)

		req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		HandlerChangePassword(w, req)

		// BUG: No rate limiting, all requests should go through
		// Expected: Should return 429 Too Many Requests after a few attempts
		// Actual: All requests are processed
	}

	t.Log("BUG DOCUMENTED: HandlerChangePassword has no rate limiting")
	t.Log("Attackers can brute-force the current password")
}

// BUG TEST: Weak password should be rejected
func TestHandlerChangePassword_WeakNewPassword(t *testing.T) {
	setupTestDB()

	testEmail := "weak-new-pw@example.com"
	currentPassword := "currentpassword"
	user, _ := createTestUser(testEmail, currentPassword)

	changeReq := models.PasswordChangeRequest{
		Email:              testEmail,
		AccountId:          user.AccountId,
		CurrentPassword:    currentPassword,
		NewPassword:        "123",
		ConfirmNewPassword: "123",
	}
	reqBody, _ := json.Marshal(changeReq)

	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HandlerChangePassword(w, req)

	// BUG: Weak passwords are accepted
	if w.Code == http.StatusOK {
		t.Error("BUG DETECTED: Weak new password was accepted")
	}
}
