package handlers

import (
	"encoding/json"
	"net/http"
	"sort"

	"omni/src/db"
	"omni/src/db/services"
	"omni/src/utils"
)

// HandlerAccountActivity returns the account's recent security activity.
// Today this is derived from the session store (sign-ins per device); a
// persistent audit log (password changes, failed attempts, profile edits)
// will extend it later.
func HandlerAccountActivity(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Same self-only rule as the sessions list: token subject must match.
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

	sessions, err := db.GetUserSessions(request.Email)
	if err != nil {
		http.Error(w, "Failed to retrieve activity", http.StatusInternalServerError)
		return
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	activity := []map[string]interface{}{}
	for _, session := range sessions {
		browser := services.ParseBrowser(session.DeviceInfo)
		ip := session.IPAddress
		if ip == "" {
			ip = "127.0.0.1"
		}
		activity = append(activity, map[string]interface{}{
			"event":     "Signed in",
			"icon":      "login",
			"source":    browser,
			"ipAddress": ip,
			"country":   session.Country,
			"dateTime":  services.FormatSessionTime(session.CreatedAt),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"activity": activity,
	})
}
