package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"omni/src/db"
	"omni/src/db/services"
	"omni/src/models"
	"omni/src/utils"
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

	// Callers may only list their own sessions: require a valid access
	// token (bearer or cookie) whose subject matches the requested email
	accessTokenStr := utils.ExtractBearerToken(r)
	if accessTokenStr == "" {
		if cookie, err := r.Cookie("access_token"); err == nil {
			accessTokenStr = cookie.Value
		}
	}
	if accessTokenStr == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	claims, err := utils.ValidateAccessToken(accessTokenStr)
	if err != nil {
		http.Error(w, "Invalid access token", http.StatusUnauthorized)
		return
	}
	if claims.Subject != request.Email {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get all sessions for the user
	sessions, err := db.GetUserSessions(request.Email)
	if err != nil {
		http.Error(w, "Failed to retrieve user sessions", http.StatusInternalServerError)
		return
	}

	// The caller's own session is the one holding their refresh token —
	// there is no middleware that stamps a sessionID into the context.
	var currentToken string
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		currentToken = cookie.Value
	}

	// Format the sessions for the response
	activeSessions := []map[string]interface{}{}
	for _, session := range sessions {
		// Get IP from X-Forwarded-For header or remote address when session was created
		ipAddress := session.IPAddress
		if ipAddress == "" {
			ipAddress = "127.0.0.1" // Default if not available
		}
		// Parse browser info from DeviceInfo (User-Agent)
		browser := services.ParseBrowser(session.DeviceInfo)

		isCurrent := currentToken != "" && session.Token == currentToken
		lastLoginTime := services.FormatSessionTime(session.LastLoginAt)
		if isCurrent {
			lastLoginTime = "Current Session"
		}

		activeSessions = append(activeSessions, map[string]interface{}{
			"id":              session.ID,
			"browser":         browser,
			"country":         session.Country, // This should be set when creating the session
			"lastLoginAt":     lastLoginTime,
			"ipAddress":       ipAddress,
			"deviceInfo":      session.DeviceInfo,
			"isCurrentDevice": isCurrent,
		})
	}
	// Send the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"activeSessions": activeSessions,
	})
}

func HandlerCheckSession(w http.ResponseWriter, r *http.Request) {
	var user *models.User

	slog.Debug("Starting session check")

	// Try Authorization header first (mobile), then fall back to cookie (web)
	accessTokenStr := utils.ExtractBearerToken(r)
	if accessTokenStr == "" {
		accessTokenCookie, cookieErr := r.Cookie("access_token")
		if cookieErr != nil {
			slog.Debug("Access token not found", "error", cookieErr)
		} else {
			accessTokenStr = accessTokenCookie.Value
		}
	}

	if accessTokenStr == "" {
		slog.Debug("No access token available")
	} else {
		// Access token exists, try to validate it
		claims, err := utils.ValidateAccessToken(accessTokenStr)
		if err != nil {
			slog.Debug("Access token validation failed", "error", err)
		} else {
			// Access token is valid, get the user
			user, err = db.GetUser(claims.Subject)
			if err != nil {
				slog.Warn("Failed to get user from access token", "error", err)
			} else {
				slog.Debug("User retrieved from access token")
			}
		}
	}

	// If we don't have a valid user at this point, try the refresh token
	if user == nil {
		slog.Debug("Attempting to use refresh token")
		// Check request body for refresh token (mobile), then cookie (web)
		var refreshTokenValue string
		var bodyReq struct {
			RefreshToken string `json:"refreshToken"`
		}
		// Try reading from body without consuming it for mobile clients
		if r.Body != nil {
			bodyBytes, _ := io.ReadAll(r.Body)
			if len(bodyBytes) > 0 {
				json.Unmarshal(bodyBytes, &bodyReq)
				refreshTokenValue = bodyReq.RefreshToken
			}
		}
		if refreshTokenValue == "" {
			refreshTokenCookie, cookieErr := r.Cookie("refresh_token")
			if cookieErr != nil {
				slog.Debug("Refresh token not found", "error", cookieErr)
				http.Error(w, "No valid session found", http.StatusUnauthorized)
				return
			}
			refreshTokenValue = refreshTokenCookie.Value
		}

		slog.Debug("Refresh token found")

		// Get the refresh token info
		tokenInfo, err := db.GetRefreshToken(refreshTokenValue)
		if err != nil {
			slog.Warn("Invalid refresh token", "error", err)
			http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
			return
		}

		slog.Debug("Refresh token info retrieved")

		// Get the user associated with this refresh token
		user, err = db.GetUser(tokenInfo.UserEmail)
		if err != nil {
			slog.Warn("User not found from refresh token", "error", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		slog.Debug("User retrieved from refresh token")

		// Generate new access token
		newAccessToken, err := utils.GenerateAccessToken(user.Email)
		if err != nil {
			slog.Error("Error generating new access token", "error", err)
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

		slog.Debug("New access token set")

		// Update the session's last login time
		sessions, err := db.GetUserSessions(user.Email)
		if err == nil {
			for _, session := range sessions {
				if session.Token == refreshTokenValue {
					db.UpdateSessionLastLogin(session.ID)
					slog.Debug("Session last login time updated")
					break
				}
			}
		} else {
			slog.Error("Failed to get user sessions", "error", err)
		}
	}

	if user == nil {
		slog.Debug("No valid user found after all checks")
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
	slog.Debug("Session check completed successfully")
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
				slog.Error("Failed to delete session", "sessionId", session.ID, "error", err)
				continue
			}
			if err := db.DeleteRefreshToken(session.Token); err != nil {
				slog.Error("Failed to delete refresh token for session", "sessionId", session.ID, "error", err)
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
		slog.Error("Failed to delete refresh token for session", "sessionId", sessionID, "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Session successfully logged out",
	})
}
