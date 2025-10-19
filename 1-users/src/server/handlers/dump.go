package handlers

import (
	"fmt"
	"io"
	"net/http"
)

func HandlerDump(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	fmt.Printf("%s\n", body)
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
