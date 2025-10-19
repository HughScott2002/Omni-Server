package main

import (
	"log"
	"net/http"
	"os"

	"example.com/transactions/v1/src/db"
	"example.com/transactions/v1/src/server"
)

func main() {
	// Initialize database
	if err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	log.Println("Transaction service starting...")

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083" // Default port for transaction service
	}

	// Create router
	router := server.Router()

	// Start server
	addr := ":" + port
	log.Printf("Transaction service listening on %s", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
