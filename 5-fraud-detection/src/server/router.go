package server

import (
	"log"
	"net/http"
	"os"

	"omni/fraud-detection/src/server/handlers"
)

// enableCORS middleware to handle CORS
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SetupRouter configures all routes for the fraud detection service
func SetupRouter() {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", handlers.HandlerHealth)

	// Risk assessment endpoint
	mux.HandleFunc("/api/fraud-detection/assess", handlers.HandlerAssessRisk)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	// Wrap with CORS middleware
	handler := enableCORS(mux)

	log.Printf("Fraud Detection Service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
