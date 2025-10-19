package handlers

import (
	"encoding/json"
	"net/http"

	"example.com/m/v2/src/db"
	"github.com/go-chi/chi/v5"
)

func GetWallet(w http.ResponseWriter, r *http.Request) {
	walletId := chi.URLParam(r, "walletId")
	wallet, err := db.GetWallet(walletId)
	if err != nil {
		http.Error(w, "Wallet not found", http.StatusNotFound)
		return
	}

	// // Get associated virtual cards
	// cards, err := db.GetVirtualCardsByWalletId(walletId)
	// if err != nil {
	// 	cards = []*models.VirtualCard{} // Empty array if no cards or error
	// }

	// // Mask card numbers in response
	// for _, card := range cards {
	// 	card.CardNumber = card.MaskedCardNumber()
	// }

	// // Create response with wallet and cards
	// response := map[string]interface{}{
	// 	"wallet": wallet,
	// 	"cards":  cards,
	// }

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wallet)
}
