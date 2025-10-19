package main

import (
	"log"
	"omni/fraud-detection/src/server"
)

func main() {
	log.Println("Starting Fraud Detection Service...")
	server.SetupRouter()
}
