package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"example.com/m/v2/src/db"
	"example.com/m/v2/src/events"
	"example.com/m/v2/src/events/consumer"
	"example.com/m/v2/src/server"
)

func main() {
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
			log.Println("Shutting down gracefully...")
			cancel()
		}()

		// Start the consumer in a goroutine
		go func() {
			log.Println("Starting Kafka consumer...")
			if err := consumer.ConsumeAccountCreatedEvents(ctx); err != nil {
				log.Printf("Kafka consumer error: %v", err)
			}
		}()
	} else {
		panic("Kafka isn't alive")
	}
	//Then branch out into go routines for the server and the event handlers

	log.Println("Wallet server is running on Port 8080")
	log.Fatal(http.ListenAndServe(":8080", server.Router()))
}
