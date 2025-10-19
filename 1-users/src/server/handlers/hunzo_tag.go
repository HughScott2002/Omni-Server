package handlers

import (
	"encoding/json"
	"net/http"

	"omni/src/db"
	"omni/src/server/middleware"
	"github.com/go-chi/chi/v5"
)

// HandlerSearchByOmniTag searches for a user by their Omni Tag
// Returns only OmniTag before contact acceptance, full info after acceptance
func HandlerSearchByOmniTag(w http.ResponseWriter, r *http.Request) {
	omniTag := chi.URLParam(r, "omnitag")

	if omniTag == "" {
		http.Error(w, "Omni Tag is required", http.StatusBadRequest)
		return
	}

	user, err := db.GetUserByOmniTag(omniTag)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get current user's account ID from session/auth context (optional)
	currentUserAccountID, _ := middleware.GetAccountIDFromContext(r)

	// Base response with only OmniTag (before contact acceptance)
	userInfo := map[string]interface{}{
		"accountId": user.AccountId,
		"omniTag":   user.OmniTag,
	}

	// Check if they are already accepted contacts
	if currentUserAccountID != "" {
		contacts, err := db.GetContactsByUser(currentUserAccountID)
		if err == nil {
			for _, contact := range contacts {
				if contact.AccountID == user.AccountId && contact.IsAccepted {
					// They are contacts - show full info
					userInfo["firstName"] = user.FirstName
					userInfo["lastName"] = user.LastName
					userInfo["email"] = user.Email
					userInfo["isContact"] = true
					break
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userInfo)
}
