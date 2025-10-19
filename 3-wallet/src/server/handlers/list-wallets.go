package handlers

import (
	"encoding/json"
	"net/http"

	"example.com/m/v2/src/db"
	"github.com/go-chi/chi/v5"
)

func ListWallets(w http.ResponseWriter, r *http.Request) {
	accountId := chi.URLParam(r, "accountId")
	wallet, err := db.ListWallets(accountId)
	if err != nil {
		http.Error(w, "No Wallets Found", http.StatusNotFound)
		return
	}

	// // Create response with each wallet and its associated cards
	// type WalletWithCards struct {
	// 	Wallet *models.Wallet          `json:"wallet"`
	// 	Cards  []*models.VirtualCard   `json:"cards"`
	// }

	// response := make([]WalletWithCards, 0, len(wallets))

	// for _, wallet := range wallets {
	// 	// Get cards for this wallet
	// 	cards, err := db.GetVirtualCardsByWalletId(wallet.WalletId)
	// 	if err != nil {
	// 		cards = []*models.VirtualCard{} // Empty array if no cards or error
	// 	}

	// 	// Mask card numbers
	// 	for _, card := range cards {
	// 		card.CardNumber = card.MaskedCardNumber()
	// 	}

	// 	response = append(response, WalletWithCards{
	// 		Wallet: wallet,
	// 		Cards:  cards,
	// 	})
	// }

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wallet)
}
