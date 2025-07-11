package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"throwback-chat/internal/db"
	"throwback-chat/internal/web"
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

	// Initialize database
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize web server
	server := web.NewServer(database, dbPath)
	router := server.SetupRouter()

	log.Printf("Starting server on %s:%s", host, port)
	log.Printf("Using database: %s", dbPath)

	if err := http.ListenAndServe(host+":"+port, router); err != nil {
		log.Fatal(err)
	}
}
