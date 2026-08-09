package main

import (
	"log/slog"
	"net/http"
	"os"

	"omni/src/db"
	"omni/src/server"
	"omni/src/server/handlers"
	"omni/src/utils"

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
	utils.InitLogger("user-service")

	// Initialize
	err := db.Init()
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	r := chi.NewRouter()

	// Kafka test producer - comment out if Kafka is not available
	// go producer.TestProducer()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://omniui-plum.vercel.app/", "http://localhost:3000", "http://localhost:5173", "http://127.0.0.1:3000", "http://127.0.0.1:5173"}, // Allow common dev origins
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true, // Required for credentials: "include"
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
	slog.Info("User server is running", "port", 8080)
	if err := http.ListenAndServe(":8080", r); err != nil {
		slog.Error("Server stopped", "error", err)
		os.Exit(1)
	}

}
func HandlerPlaceHolder(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{})
}
