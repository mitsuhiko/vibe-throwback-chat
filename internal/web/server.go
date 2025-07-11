package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"throwback-chat/internal/db"
)

type Server struct {
	db        *db.DB
	dbPath    string
	wsHandler *WebSocketHandler
}

func NewServer(database *db.DB, dbPath string) *Server {
	return &Server{
		db:        database,
		dbPath:    dbPath,
		wsHandler: NewWebSocketHandler(database),
	}
}

func (s *Server) SetupRouter() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Routes
	r.Get("/api/health", s.handleHealth)
	r.Get("/ws", s.handleWebSocket)

	return r
}
