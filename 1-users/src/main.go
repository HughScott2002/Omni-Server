package main

import (
	"fmt"
	"log"
	"net/http"

	"omni/src/db"
	"omni/src/server"
	"omni/src/server/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

//TODO: Implement rate limiting on login attempts to prevent brute-force attacks.

//JWT
//TODO: Use strong, randomly generated secrets for signing JWTs
//TODO: Include essential claims like 'exp' (expiration), 'iat' (issued at), and 'jti' (JWT ID).
//TODO: Keep JWT payload minimal to reduce token size.

//TODO: ADD PROGRESS TRACKER FOR

func main() {
	// Initialize
	err := db.Init()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	r := chi.NewRouter()

	// Kafka test producer - comment out if Kafka is not available
	// go producer.TestProducer()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // Allow your frontend origin
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Routing
	// Everything in the service needs to start with /api/users to be properly routed
	r.Route("/api/users", func(r chi.Router) {
		r.Mount("/auth", server.Router())
		r.Post("/dump", handlers.HandlerDump)
		r.Get("/health", handlers.HandlerHealth)
		// r.Route("/account", func(r chi.Router) {
		// 	r.Get("/update", HandlerPlaceHolder)
		// })
	})
	fmt.Println("User server is running on Port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))

}

func HandlerPlaceHolder(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{})
}
