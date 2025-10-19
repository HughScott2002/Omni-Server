package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"omni/src/db"
	"omni/src/db/services"
	"omni/src/models"
	"omni/src/utils"
	"github.com/go-chi/chi/v5"
)

func HandlerListActiveSessions(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse the request body
	var request struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate that email is provided
	if request.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	// Get all sessions for the user
	sessions, err := db.GetUserSessions(request.Email)
	if err != nil {
		http.Error(w, "Failed to retrieve user sessions", http.StatusInternalServerError)
		return
	}

	// Format the sessions for the response
	var activeSessions []map[string]interface{}
	for _, session := range sessions {
		// Get IP from X-Forwarded-For header or remote address when session was created
		ipAddress := session.IPAddress
		if ipAddress == "" {
			ipAddress = "127.0.0.1" // Default if not available
		}
		// Parse browser info from DeviceInfo (User-Agent)
		browser := services.ParseBrowser(session.DeviceInfo)

		// Format the time
		lastLoginTime := services.FormatSessionTime(session.LastLoginAt)

		activeSessions = append(activeSessions, map[string]interface{}{
			"id":              session.ID,
			"browser":         browser,
			"country":         session.Country, // This should be set when creating the session
			"lastLoginAt":     lastLoginTime,
			"ipAddress":       ipAddress,
			"deviceInfo":      session.DeviceInfo,
			"isCurrentDevice": false, // Will be set to true for current session
		})
	}
	var currentSessionID string
	if sid := r.Context().Value("sessionID"); sid != nil {
		currentSessionID = sid.(string)
	}
	// Only try to mark current session if we have a session ID
	if currentSessionID != "" {
		for i, session := range activeSessions {
			if session["id"] == currentSessionID {
				activeSessions[i]["isCurrentDevice"] = true
				activeSessions[i]["lastLoginAt"] = "Current Session"
			}
		}
	}
	// Send the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"activeSessions": activeSessions,
	})
}

func HandlerCheckSession(w http.ResponseWriter, r *http.Request) {
	var user *models.User
	var err error

	log.Println("Starting session check")

	// Try to get the access token
	accessTokenCookie, err := r.Cookie("access_token")
	if err != nil {
		log.Println("Access token not found:", err)
	} else {
		// Access token exists, try to validate it
		claims, err := utils.ValidateAccessToken(accessTokenCookie.Value)
		if err != nil {
			log.Println("Access token validation failed:", err)
		} else {
			// Access token is valid, get the user
			user, err = db.GetUser(claims.Subject)
			if err != nil {
				log.Println("Failed to get user from access token:", err)
			} else {
				log.Println("User retrieved from access token")
			}
		}
	}

	// If we don't have a valid user at this point, try the refresh token
	if user == nil {
		log.Println("Attempting to use refresh token")
		refreshTokenCookie, err := r.Cookie("refresh_token")
		if err != nil {
			log.Println("Refresh token not found:", err)
			http.Error(w, "No valid session found", http.StatusUnauthorized)
			return
		}

		log.Printf("Refresh token found: %s", refreshTokenCookie.Value)

		// Get the refresh token info
		tokenInfo, err := db.GetRefreshToken(refreshTokenCookie.Value)
		if err != nil {
			log.Println("Invalid refresh token:", err)
			http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
			return
		}

		log.Printf("Refresh token info retrieved: %+v", tokenInfo)

		// Get the user associated with this refresh token
		user, err = db.GetUser(tokenInfo.UserEmail)
		if err != nil {
			log.Println("User not found from refresh token:", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		log.Println("User retrieved from refresh token")

		// Generate new access token
		newAccessToken, err := utils.GenerateAccessToken(user.Email)
		if err != nil {
			log.Println("Error generating new access token:", err)
			http.Error(w, "Error generating new access token", http.StatusInternalServerError)
			return
		}

		// Set new access token cookie
		// http.SetCookie(w, &http.Cookie{
		// 	Name:     "access_token",
		// 	Value:    newAccessToken,
		// 	HttpOnly: false,
		// 	Secure:   false,
		// 	Path:     "/",

		// 	SameSite: http.SameSiteStrictMode,
		// 	MaxAge:   900, // 15 minutes
		// })
		utils.SetCookie(w, "access_token", newAccessToken, 15*60) // 15 minutes
		// utils.SetCookie(w, "refresh_token", refreshToken, 7*24*60*60)

		log.Println("New access token set")

		// Update the session's last login time
		sessions, err := db.GetUserSessions(user.Email)
		if err == nil {
			for _, session := range sessions {
				if session.Token == refreshTokenCookie.Value {
					db.UpdateSessionLastLogin(session.ID)
					log.Println("Session last login time updated")
					break
				}
			}
		} else {
			log.Println("Failed to get user sessions:", err)
		}
	}

	if user == nil {
		log.Println("No valid user found after all checks")
		http.Error(w, "No valid session found", http.StatusUnauthorized)
		return
	}

	// Prepare the response
	userData := map[string]interface{}{
		"user": map[string]string{
			"id":        user.AccountId,
			"email":     user.Email,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
			"kycStatus": user.KYCStatus.String(),
		},
	}

	// Send the response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userData)
	log.Println("Session check completed successfully")
}
func HandlerLogoutAllOtherSessions(w http.ResponseWriter, r *http.Request) {
	// Get current refresh token
	refreshTokenCookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "No active session found", http.StatusUnauthorized)
		return
	}

	// Get token info to get user email
	tokenInfo, err := db.GetRefreshToken(refreshTokenCookie.Value)
	if err != nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	// Get all user sessions
	sessions, err := db.GetUserSessions(tokenInfo.UserEmail)
	if err != nil {
		http.Error(w, "Failed to retrieve user sessions", http.StatusInternalServerError)
		return
	}

	// Delete all sessions except current one
	for _, session := range sessions {
		if session.Token != refreshTokenCookie.Value {
			if err := db.DeleteSession(session.ID); err != nil {
				log.Printf("Failed to delete session %s: %v", session.ID, err)
				continue
			}
			if err := db.DeleteRefreshToken(session.Token); err != nil {
				log.Printf("Failed to delete refresh token for session %s: %v", session.ID, err)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Successfully logged out all other devices",
	})
}

func HandlerLogoutSessionById(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionid")
	if sessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	session, err := db.GetSession(sessionID)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Delete session and its refresh token
	if err := db.DeleteSession(sessionID); err != nil {
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	if err := db.DeleteRefreshToken(session.Token); err != nil {
		log.Printf("Failed to delete refresh token for session %s: %v", sessionID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Session successfully logged out",
	})
}
