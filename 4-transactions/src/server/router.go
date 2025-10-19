package server

import (
	"log"
	"net/http"

	"example.com/transactions/v1/src/server/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func Router() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Debug middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Route Debug - Method: %s, Path: %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Transaction service is healthy"))
	})

	// Transaction routes
	r.Route("/api/transactions", func(r chi.Router) {
		// Transfer money between wallets
		r.Post("/transfer", handlers.HandlerTransferMoney)

		// Card purchases
		r.Post("/purchase", handlers.HandlerCardPurchase)

		// Get transaction history
		r.Get("/account/{accountId}", handlers.HandlerGetTransactionsByAccount)
		r.Get("/wallet/{walletId}", handlers.HandlerGetTransactionsByWallet)

		// Get specific transaction
		r.Get("/{transactionId}", handlers.HandlerGetTransaction)
	})

	// 404 handler
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("404 Not Found - Method: %s, Path: '%s'", r.Method, r.URL.Path)
		http.Error(w, "Route not found", http.StatusNotFound)
	})

	return r
}
