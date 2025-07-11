package web

import (
	"encoding/json"
	"log"
	"net/http"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/db"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WebSocket command message types
type WSRequest struct {
	Type  string `json:"type,omitempty"`
	Cmd   string `json:"cmd,omitempty"`
	ReqID string `json:"req_id,omitempty"`
}

type WSResponse struct {
	Type  string      `json:"type"`
	ReqID string      `json:"req_id"`
	Okay  bool        `json:"okay"`
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

func NewWSResponse(reqID string, okay bool, err string, data interface{}) WSResponse {
	return WSResponse{
		Type:  "response",
		ReqID: reqID,
		Okay:  okay,
		Error: err,
		Data:  data,
	}
}

func ParseWSMessage(data []byte) (*WSRequest, error) {
	var msg WSRequest
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

// DecodeWSData helper that terminates websocket on failure
func DecodeWSData(sess *chat.Session, data []byte, reqID string, target interface{}) error {
	if err := json.Unmarshal(data, target); err != nil {
		// Send error response
		sess.RespondError(reqID, "Invalid request format")
		// Close the connection by returning a special error
		return &websocketTerminateError{message: "Invalid JSON in request"}
	}
	return nil
}

// Custom error type to signal websocket termination
type websocketTerminateError struct {
	message string
}

func (e *websocketTerminateError) Error() string {
	return e.message
}

type WebSocketHandler struct {
	db       *db.DB
	sessions *chat.SessionManager
}

func NewWebSocketHandler(database *db.DB) *WebSocketHandler {
	return &WebSocketHandler{
		db:       database,
		sessions: chat.NewSessionManager(),
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	s.wsHandler.HandleConnection(w, r)
}

func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	sessionID := uuid.New().String()
	session := h.sessions.AddSession(sessionID, conn)

	defer func() {
		h.sessions.RemoveSession(sessionID)
		conn.Close()
	}()

	log.Printf("WebSocket connection established: %s", sessionID)

	for {
		_, messageData, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Failed to read message from %s: %v", sessionID, err)
			break
		}

		h.sessions.UpdateHeartbeat(sessionID)

		if err := h.handleMessage(session, messageData); err != nil {
			// Check if this is a websocket termination error
			if _, isTerminate := err.(*websocketTerminateError); isTerminate {
				log.Printf("Terminating websocket connection %s: %v", sessionID, err)
				break
			}
			log.Printf("Failed to handle message from %s: %v", sessionID, err)
		}
	}
}

func (h *WebSocketHandler) handleMessage(sess *chat.Session, data []byte) error {
	msg, err := ParseWSMessage(data)
	if err != nil {
		sess.RespondError("", "Invalid JSON")
		return &websocketTerminateError{message: "Invalid JSON in message"}
	}

	log.Printf("Received command: %s from session %s", msg.Cmd, sess.ID)

	switch msg.Cmd {
	case "login":
		return h.HandleLogin(sess, data)
	case "logout":
		return h.HandleLogout(sess, data)
	case "heartbeat":
		return h.HandleHeartbeat(sess, msg.ReqID)
	case "join":
		return h.HandleJoin(sess, data)
	case "leave":
		return h.HandleLeave(sess, data)
	case "message":
		return h.HandleMessage(sess, data)
	default:
		return sess.RespondError(msg.ReqID, "Unknown command")
	}
}
