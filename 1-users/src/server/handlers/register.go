package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"omni/src/db"
	"omni/src/db/services"
	"omni/src/events/producer"
	"omni/src/models"
	"omni/src/models/events"
	"omni/src/utils"
	"golang.org/x/crypto/bcrypt"
)

//TODO: Need a way to add Tags
//TODO: Need to know what kind of account it is? Personal? Business? Ect.

func HandlerRegister(w http.ResponseWriter, r *http.Request) {
	//Change
	deviceInfo := r.Header.Get("User-Agent")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	fmt.Println(deviceInfo)
	fmt.Printf("%s\n", body)
	//

	var user models.User
	// Decode the JSON body into the User struct
	if err := json.Unmarshal(body, &user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate Omni Tag
	if err := utils.ValidateOmniTag(user.OmniTag); err != nil {
		http.Error(w, fmt.Sprintf("Invalid Omni Tag: %v", err), http.StatusBadRequest)
		return
	}

	// Check if Omni Tag already exists
	tagExists, err := db.OmniTagExists(user.OmniTag)
	if err != nil {
		http.Error(w, "Error checking Omni Tag availability", http.StatusInternalServerError)
		return
	}
	if tagExists {
		http.Error(w, "Omni Tag already taken", http.StatusConflict)
		return
	}

	// // Check if the user already exists
	// if _, exists := db.Users[user.Email]; exists {
	// 	utils.ErrorResponse(w, "User already exists", 500)
	// 	// http.Error(w, "User already exists", http.StatusConflict)
	// 	return
	// }
	//Create the Account Id
	user.AccountId, err = utils.GenerateAccountId()
	if err != nil {
		http.Error(w, "Error generating account ID", http.StatusInternalServerError)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.UnHashedPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	user.HashedPassword = string(hashedPassword)

	// Set the initial KYC status to Pending
	user.KYCStatus = models.KYCStatusPending

	// Save the user
	err = db.AddUser(&user)
	if err != nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	userCreatedEvent := events.AccountCreatedEvent{
		AccountId: user.AccountId,
		Currency:  user.Currency,
		KYCStatus: user.KYCStatus,
	}

	err = producer.ProduceAccountCreatedEvent(userCreatedEvent)
	if err != nil {
		log.Printf("failed to produce user created event: %v", err)
	}
	log.Printf("KAKFA EVENT account-created sent acc#: %s", userCreatedEvent.AccountId)

	// Generate access token
	accessToken, err := utils.GenerateAccessToken(user.Email)
	if err != nil {
		http.Error(w, "Error generating access token", http.StatusInternalServerError)
		return
	}
	// Generate refresh token
	refreshToken, err := utils.GenerateRefreshToken(user.Email)
	if err != nil {
		http.Error(w, "Error generating refresh token", http.StatusInternalServerError)
		return
	}

	// Create session
	session, err := services.CreateUserSession(r, &user, refreshToken)
	if err != nil {
		http.Error(w, "Error creating session", http.StatusInternalServerError)
		return
	}

	// Set cookies
	// http.SetCookie(w, &http.Cookie{
	// 	Name:     "access_token",
	// 	Value:    accessToken,
	// 	HttpOnly: false,
	// 	Secure:   false,
	// 	Path:     "/",

	// 	SameSite: http.SameSiteStrictMode,
	// 	MaxAge:   900, // 15 minutes
	// })

	// http.SetCookie(w, &http.Cookie{
	// 	Name:     "refresh_token",
	// 	Value:    refreshToken,
	// 	HttpOnly: false,
	// 	Secure:   false,
	// 	Path:     "/",

	// 	SameSite: http.SameSiteStrictMode,
	// 	MaxAge:   604800, // 7 days
	// })
	utils.SetCookie(w, "access_token", accessToken, 15*60) // 15 minutes
	utils.SetCookie(w, "refresh_token", refreshToken, 7*24*60*60)
	// // Create a response object
	// response := map[string]string{
	// 	"message":   "User registered successfully",
	// 	"kycstatus": user.KYCStatus.String(),
	// 	"accountId": user.AccountId,
	// }
	// // Set headers and return the response
	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusCreated)
	// // w.Write(users[user.Email])
	// json.NewEncoder(w).Encode(response)
	// After successful registration
	userData := map[string]interface{}{
		"user": map[string]string{
			"id":        user.AccountId,
			"email":     user.Email,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
			"omniTag":   user.OmniTag,
			"kycStatus": user.KYCStatus.String(),
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
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(userData)
}
