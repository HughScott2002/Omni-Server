package main

import (
	"log/slog"

	"omni/fraud-detection/src/server"
	"omni/fraud-detection/src/utils"
)

func main() {
	utils.InitLogger("fraud-detection-service")

	slog.Info("Starting Fraud Detection Service...")
	server.SetupRouter()
}
