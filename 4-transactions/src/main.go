package main

import (
	"log/slog"
	"net/http"
	"os"

	"example.com/transactions/v1/src/db"
	"example.com/transactions/v1/src/server"
	"example.com/transactions/v1/src/utils"
)

func main() {
	utils.InitLogger("transaction-service")

	// Initialize database
	if err := db.Init(); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	slog.Info("Transaction service starting...")

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083" // Default port for transaction service
	}

	// Create router
	router := server.Router()

	// Start server
	addr := ":" + port
	slog.Info("Transaction service listening on", "addr", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
