package server

import (
	"net/http"

	"omni/src/server/handlers"
	"omni/src/server/middleware"
	"github.com/go-chi/chi/v5"
)

// TODO: FIX THE AUTH, it needs to accept tokens on each request and make sure its vaild
// TODO: Fix the rate limiter
func Router() http.Handler {
	r := chi.NewRouter()

	// r.Post("/update", handlers.HandlerUpdateUser)

	// Protected routes
	r.Group(func(r chi.Router) {
		// r.Use(middleware.RequireAuth)  // Assuming you have this middleware
		r.Get("/health", handlers.HandlerHealth)
		r.Get("/check-session", handlers.HandlerCheckSession)
		r.Post("/refresh", middleware.RateLimitMiddleware(handlers.HandlerRefreshToken))
		// r.Get("/list-sessions", middleware.RateLimitMiddleware(handlers.HandlerListActiveSessions))

		// User account management
		r.Route("/account", func(r chi.Router) {
			r.Post("/login", handlers.HandlerLogin)
			r.Post("/register", handlers.HandlerRegister)
			r.Post("/logout", handlers.HandlerLogout)
			r.Get("/{accountid}", handlers.HandlerGetUserProfile)
			r.Put("/{accountid}", handlers.HandlerUpdateUserProfile)
			r.Delete("/{accountid}", handlers.HandlerDeleteUserAccount)
			r.Post("/change-password", handlers.HandlerChangePassword)
		})

		// Omni Tag search
		r.Get("/search/omnitag/{omnitag}", handlers.HandlerSearchByOmniTag)

		// Contact management - all routes require authentication
		r.Route("/contacts", func(r chi.Router) {
			r.Use(middleware.RequireAuth)

			// Send a contact request (by OmniTag)
			r.Post("/request", handlers.HandlerSendContactRequest)

			// Get all contacts for a user
			r.Get("/{accountid}", handlers.HandlerGetContacts)

			// Get pending contact requests (received)
			r.Get("/{accountid}/pending", handlers.HandlerGetPendingRequests)

			// Get sent contact requests
			r.Get("/{accountid}/sent", handlers.HandlerGetSentRequests)

			// Accept a contact request
			r.Post("/{contactid}/accept", handlers.HandlerAcceptContactRequest)

			// Reject a contact request
			r.Post("/{contactid}/reject", handlers.HandlerRejectContactRequest)

			// Block a contact
			r.Post("/{contactid}/block", handlers.HandlerBlockContact)

			// Delete a contact
			r.Delete("/{contactid}", handlers.HandlerDeleteContact)
		})

		// KYC management
		r.Route("/kyc", func(r chi.Router) {
			r.Get("/{accountid}/status", handlers.HandlerGetKYCStatus)
			r.Post("/{accountid}/submit", handlers.HandlerSubmitKYC)
			r.Post("/{accountid}/approve", handlers.HandlerApproveKYC)
			r.Post("/{accountid}/reject", handlers.HandlerRejectKYC)
		})

		// Security settings
		r.Route("/security", func(r chi.Router) {

			//Session management
			r.Route("/sessions", func(r chi.Router) {
				// All sessions currently logged in
				r.Post("/", handlers.HandlerListActiveSessions)
				//Logout for all sessions that isn't the one the user is using right now
				r.Post("/logout-others", handlers.HandlerLogoutAllOtherSessions)
				//TODO: ADD Logout for individual sessions
				r.Post("/logout/{sessionid}", handlers.HandlerLogoutSessionById)
			})

			// Two-factor authentication on these devices has been remembered for 30 days.

			//TODO: ADD A Activity history logger
			r.Route("/2fa", func(r chi.Router) {
				// The last 30 days of activity on your account:
				// Event	       Source	        IP address	 Date and time	    Country
				// Log in failure  Chrome (Linux)	127.0.0.2	 Jan 2, 12:22 PM	United States

				// Two-factor authentication (2FA) helps accounts secure by adding an extra layer of protection beyond a password.
				// By default, we require you to set up a 2FA app that can generate 2FA codes,
				// but you can add a security key to log in even quicker.
				r.Post("/enable", handlers.HandlerEnable2FA)
				r.Post("/disable", handlers.HandlerDisable2FA)
			})

		})
	})

	return r
}

func HandlerPlaceHolder(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{})
}
