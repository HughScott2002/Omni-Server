package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"example.com/m/v2/src/db"
	"example.com/m/v2/src/events"
	"example.com/m/v2/src/events/consumer"
	"example.com/m/v2/src/server"
	"example.com/m/v2/src/utils"
)

func main() {
	utils.InitLogger("wallet-service")

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//Database Init
	err := db.Init()
	if err != nil {
		panic("Could not init Database")
	}

	//Kafka Init
	kakfaIsAlive := events.KafkaInit(ctx)
	if kakfaIsAlive {
		// Handle shutdown gracefully
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			<-sigChan
			slog.Info("Shutting down gracefully...")
			cancel()
		}()

		// Start the consumers in goroutines
		go func() {
			slog.Info("Starting Kafka consumer...")
			if err := consumer.ConsumeAccountCreatedEvents(ctx); err != nil {
				slog.Error("Kafka consumer error", "error", err)
			}
		}()

		go func() {
			if err := consumer.ConsumeKYCApprovedEvents(ctx); err != nil {
				slog.Error("Kafka kyc-approved consumer error", "error", err)
			}
		}()
	} else {
		panic("Kafka isn't alive")
	}
	//Then branch out into go routines for the server and the event handlers

	slog.Info("Wallet server is running", "port", 8080)
	if err := http.ListenAndServe(":8080", server.Router()); err != nil {
		slog.Error("Server stopped", "error", err)
		os.Exit(1)
	}
}
