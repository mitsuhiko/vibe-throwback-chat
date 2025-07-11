package web

import (
	"net/http"

	"throwback-chat/internal/utils"
)

type HealthResponse struct {
	Status string `json:"status"`
	DBPath string `json:"db_path"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "ok",
		DBPath: s.dbPath,
	}
	utils.SendJSON(w, response)
}
