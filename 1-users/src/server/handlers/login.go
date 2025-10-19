package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"omni/src/db"
	"omni/src/db/services"
	"omni/src/models"
	"omni/src/utils"
	"golang.org/x/crypto/bcrypt"
)

func HandlerLogin(w http.ResponseWriter, r *http.Request) {
	// deviceInfo := r.Header.Get("User-Agent")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	fmt.Printf("%s\n", body)

	var loginRequest models.User

	// Decode the JSON body into the User struct
	if err := json.Unmarshal(body, &loginRequest); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	storedUser, err := db.GetUser(loginRequest.Email)
	if err != nil {
		http.Error(w, "User doesn't exist", http.StatusNotFound)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedUser.HashedPassword), []byte(loginRequest.UnHashedPassword))
	if err != nil {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	// Generate tokens
	accessToken, err := utils.GenerateAccessToken(loginRequest.Email)
	if err != nil {
		http.Error(w, "Error generating access token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(loginRequest.Email)
	if err != nil {
		http.Error(w, "Error generating refresh token", http.StatusInternalServerError)
		return
	}

	// Create session
	session, err := services.CreateUserSession(r, storedUser, refreshToken)
	if err != nil {
		http.Error(w, "Error creating session", http.StatusInternalServerError)
		return
	}

	// Set cookies
	utils.SetCookie(w, "access_token", accessToken, 15*60)        // 15 minutes
	utils.SetCookie(w, "refresh_token", refreshToken, 7*24*60*60) // 7 days

	userData := map[string]interface{}{
		"user": map[string]string{
			"id":        storedUser.AccountId,
			"email":     storedUser.Email,
			"firstName": storedUser.FirstName,
			"lastName":  storedUser.LastName,
			"kycStatus": storedUser.KYCStatus.String(),
			// "account":   storedUser.Status.String(),
		},
		"session": map[string]interface{}{
			"id":         session.ID,
			"browser":    session.Browser,
			"ipAddress":  session.IPAddress,
			"deviceInfo": session.DeviceInfo,
			// "refreshToken": refreshToken,
			// "accessToken":  accessToken,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userData)
}
