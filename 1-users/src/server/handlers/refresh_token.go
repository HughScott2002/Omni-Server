package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"omni/src/db"
	"omni/src/utils"
)

func HandlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	// Get the refresh token from the cookie
	refreshTokenCookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "Refresh token not found", http.StatusUnauthorized)
		return
	}

	// Get the refresh token info from the database
	tokenInfo, err := db.GetRefreshToken(refreshTokenCookie.Value)
	if err != nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	// Get the user associated with this refresh token
	user, err := db.GetUser(tokenInfo.UserEmail)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Generate new access token
	newAccessToken, err := utils.GenerateAccessToken(user.Email)
	if err != nil {
		http.Error(w, "Error generating new access token", http.StatusInternalServerError)
		return
	}

	// Generate new refresh token
	newRefreshToken, err := utils.GenerateRefreshToken(user.Email)
	if err != nil {
		http.Error(w, "Error generating new refresh token", http.StatusInternalServerError)
		return
	}

	// Update the refresh token in the database
	err = db.DeleteRefreshToken(refreshTokenCookie.Value)
	if err != nil {
		http.Error(w, "Error deleting old refresh token", http.StatusInternalServerError)
		return
	}

	err = db.AddRefreshToken(newRefreshToken, db.RefreshTokenInfo{
		UserEmail:  user.Email,
		DeviceInfo: tokenInfo.DeviceInfo,
		CreatedAt:  time.Now(),
	})
	if err != nil {
		http.Error(w, "Error storing new refresh token", http.StatusInternalServerError)
		return
	}

	// Update the session's last login time
	sessions, err := db.GetUserSessions(user.Email)
	if err == nil {
		for _, session := range sessions {
			if session.Token == refreshTokenCookie.Value {
				db.UpdateSessionLastLogin(session.ID)
				break
			}
		}
	}

	// // Set new cookies
	// http.SetCookie(w, &http.Cookie{
	// 	Name:     "access_token",
	// 	Value:    newAccessToken,
	// 	HttpOnly: false,
	// 	Secure:   false,
	// 	SameSite: http.SameSiteStrictMode,
	// 	Path:     "/",
	// 	MaxAge:   900, // 15 minutes
	// })

	// http.SetCookie(w, &http.Cookie{
	// 	Name:     "refresh_token",
	// 	Value:    newRefreshToken,
	// 	HttpOnly: false,
	// 	Secure:   false,
	// 	SameSite: http.SameSiteStrictMode,
	// 	Path:     "/",
	// 	MaxAge: 604800, // 7 days
	// })
	utils.SetCookie(w, "access_token", newAccessToken, 15*60) // 15 minutes
	utils.SetCookie(w, "refresh_token", newRefreshToken, 7*24*60*60)
	// Send the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": map[string]string{
			"id":        user.AccountId,
			"email":     user.Email,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
			"kycStatus": user.KYCStatus.String(),
		},
	})
}
