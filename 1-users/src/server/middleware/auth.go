package middleware

import (
	"context"
	"net/http"

	"omni/src/db"
	"omni/src/utils"
)

// RequireAuth is a middleware that validates the access token from cookies
// and adds the user's account ID to the request context
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get the access token from cookies
		accessTokenCookie, err := r.Cookie("access_token")
		if err != nil {
			http.Error(w, "Unauthorized: No access token", http.StatusUnauthorized)
			return
		}

		// Validate the access token
		claims, err := utils.ValidateAccessToken(accessTokenCookie.Value)
		if err != nil {
			// Access token is invalid or expired
			http.Error(w, "Unauthorized: Invalid or expired access token", http.StatusUnauthorized)
			return
		}

		// Get the user from the database
		user, err := db.GetUser(claims.Subject)
		if err != nil {
			http.Error(w, "Unauthorized: User not found", http.StatusUnauthorized)
			return
		}

		// Check if user account is active
		if !user.IsActive() {
			http.Error(w, "Unauthorized: Account is not active", http.StatusUnauthorized)
			return
		}

		// Add user information to request context
		ctx := context.WithValue(r.Context(), "userEmail", user.Email)
		ctx = context.WithValue(ctx, "accountID", user.AccountId)
		ctx = context.WithValue(ctx, "user", user)

		// Call the next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetAccountIDFromContext retrieves the account ID from the request context
func GetAccountIDFromContext(r *http.Request) (string, bool) {
	accountID, ok := r.Context().Value("accountID").(string)
	return accountID, ok
}

// GetUserEmailFromContext retrieves the user email from the request context
func GetUserEmailFromContext(r *http.Request) (string, bool) {
	email, ok := r.Context().Value("userEmail").(string)
	return email, ok
}
