package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func HandlerRemoveDevice(w http.ResponseWriter, r *http.Request) {
	deviceId := chi.URLParam(r, "deviceId")
	// TODO: Implement logic to remove specific device
	json.NewEncoder(w).Encode(map[string]string{"message": "Remove device " + deviceId})
}

func HandlerRemoveAllDevices(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement logic to remove all devices
	json.NewEncoder(w).Encode(map[string]string{"message": "Remove all devices"})
}
