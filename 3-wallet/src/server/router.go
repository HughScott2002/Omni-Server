package server

import (
	"log"
	"net/http"

	"example.com/m/v2/src/server/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// TODO: GET A LIST OF THE WALLETS AND EACH ATTACHED CARD

func Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Add debug middleware to log route matching
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Route Debug - Method: %s, Path: %s, RoutePattern: %v", r.Method, r.URL.Path, chi.RouteContext(r.Context()))
			next.ServeHTTP(w, r)
		})
	})

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // Allow your frontend origin
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Route("/api/wallets", func(r chi.Router) {
		//Wallet routes
		r.Get("/{walletId}", handlers.GetWallet)
		r.Get("/list/{accountId}", handlers.ListWallets) //List all the wallets
		r.Get("/recover", func(http.ResponseWriter, *http.Request) { panic("foo") })

		// Virtual card routes under /api/wallets/cards
		r.Route("/cards", func(r chi.Router) {
			// Create a new virtual card
			r.Post("/", handlers.HandlerCreateVirtualCard)

			// Get all cards for an account
			r.Get("/account/{accountid}", handlers.HandlerGetVirtualCardsByAccount)

			// Card operations
			r.Route("/{cardid}", func(r chi.Router) {
				r.Get("/", handlers.HandlerGetVirtualCard)
				r.Put("/", handlers.HandlerUpdateVirtualCard)
				r.Delete("/", handlers.HandlerDeleteVirtualCard)
				r.Post("/block", handlers.HandlerBlockVirtualCard)
				r.Post("/topup", handlers.HandlerTopUpVirtualCard)
				r.Post("/request-physical", handlers.HandlerRequestPhysicalCard)
			})
		})
	})

	//TODO: When there is a new Wallet Create a new card event

	// Catch-all for debugging unmatched routes
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("404 Not Found - Method: %s, Path: '%s', Raw Path: '%s'", r.Method, r.URL.Path, r.URL.RawPath)
		http.Error(w, "Route not found", http.StatusNotFound)
	})

	return r
}
