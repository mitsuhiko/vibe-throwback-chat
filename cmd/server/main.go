package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Get configuration from environment
	port := os.Getenv("TBCHAT_PORT")
	if port == "" {
		port = "8080"
	}

	host := os.Getenv("TBCHAT_HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	dbPath := os.Getenv("TBCHAT_DB")
	if dbPath == "" {
		dbPath = "chat.db"
	}

	// Initialize router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Routes
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","db_path":"` + dbPath + `"}`))
	})

	// TODO: Add websocket endpoint
	// TODO: Add other API endpoints

	log.Printf("Starting server on %s:%s", host, port)
	log.Printf("Using database: %s", dbPath)

	if err := http.ListenAndServe(host+":"+port, r); err != nil {
		log.Fatal(err)
	}
}
