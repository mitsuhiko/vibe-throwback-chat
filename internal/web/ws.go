package web

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/db"
	"throwback-chat/internal/models"

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

// WSEvent represents a WebSocket event message
type WSEvent struct {
	Type      string `json:"type"`
	ChannelID int    `json:"channel_id"`
	Event     string `json:"event"`
	UserID    int    `json:"user_id"`
	Nickname  string `json:"nickname"`
	SentAt    string `json:"sent_at"`
}

// SessionInfoResponse represents the response data for session_info command
type SessionInfoResponse struct {
	SessionID string        `json:"session_id"`
	UserID    int           `json:"user_id"`
	Nickname  string        `json:"nickname"`
	Channels  []ChannelInfo `json:"channels"`
}

// ChannelInfo represents channel information in session_info response
type ChannelInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Topic string `json:"topic"`
}

// ChannelUsersResponse represents the response data for channel_users command
type ChannelUsersResponse struct {
	Users []models.ChannelUser `json:"users"`
}

// QuitResponse represents the response data for quit command
type QuitResponse struct {
	Message string `json:"message"`
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
		sess.RespondError(reqID, "Invalid request format", err)
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
	h := &WebSocketHandler{
		db:       database,
		sessions: chat.NewSessionManager(),
	}

	// Set up callback for expired sessions to generate leave events
	h.sessions.SetSessionExpiredCallback(h.handleExpiredSession)

	return h
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

	// Check for existing session ID in query parameters
	var sessionID string
	var session *chat.Session
	existingSessionID := r.URL.Query().Get("session_id")

	if existingSessionID != "" {
		// Try to reuse existing session
		if existingSession := h.sessions.GetSession(existingSessionID); existingSession != nil {
			log.Printf("Reusing existing session: %s", existingSessionID)
			sessionID = existingSessionID
			// Transfer the connection to the existing session
			h.sessions.TransferConnection(existingSessionID, conn)
			session = existingSession
			// Generate join events for session restoration if user was logged in
			h.generateJoinEventsForSessionRestore(session)
		} else {
			log.Printf("Requested session %s not found, creating new session", existingSessionID)
			sessionID = uuid.New().String()
			session = h.sessions.AddSession(sessionID, conn)
		}
	} else {
		// Create new session
		sessionID = uuid.New().String()
		session = h.sessions.AddSession(sessionID, conn)
	}

	defer func() {
		// Generate leave events for unexpected disconnections
		h.handleUnexpectedDisconnect(sessionID)
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
		sess.RespondError("", "Invalid JSON", err)
		return &websocketTerminateError{message: "Invalid JSON in message"}
	}

	log.Printf("Received command: %s from session %s", msg.Cmd, sess.ID)

	switch msg.Cmd {
	case "login":
		return h.HandleLogin(sess, data)
	case "logout":
		return h.HandleLogout(sess, data)
	case "quit":
		return h.HandleQuit(sess, data)
	case "session_info":
		return h.HandleSessionInfo(sess, data)
	case "heartbeat":
		return h.HandleHeartbeat(sess, msg.ReqID)
	case "join":
		return h.HandleJoin(sess, data)
	case "leave":
		return h.HandleLeave(sess, data)
	case "message":
		return h.HandleMessage(sess, data)
	case "me":
		return h.HandleMe(sess, data)
	case "nick":
		return h.HandleNick(sess, data)
	case "kick":
		return h.HandleKick(sess, data)
	case "topic":
		return h.HandleTopic(sess, data)
	case "list_channels":
		return h.HandleListChannels(sess, data)
	case "my_channels":
		return h.HandleMyChannels(sess, data)
	case "get_history":
		return h.HandleHistory(sess, data)
	case "announce":
		return h.HandleAnnounce(sess, data)
	case "channel_users":
		return h.HandleChannelUsers(sess, data)
	default:
		return sess.RespondError(msg.ReqID, "Unknown command", nil)
	}
}

// handleUnexpectedDisconnect generates leave events when a session disconnects unexpectedly
func (h *WebSocketHandler) handleUnexpectedDisconnect(sessionID string) {
	session := h.sessions.GetSession(sessionID)
	if session == nil {
		return
	}

	// Check if user was logged in
	if session.UserID == nil || session.Nickname == nil {
		// Not logged in, just remove the session normally
		h.sessions.RemoveSession(sessionID)
		return
	}

	userID := *session.UserID
	nickname := *session.Nickname
	channels := session.GetChannels()

	log.Printf("Generating leave events for unexpected disconnect of user %s (ID: %d)", nickname, userID)

	// Send leave events to all channels the user was in
	for _, channelID := range channels {
		// Create database record
		_, err := models.CreateMessage(h.db, &channelID, userID, "connection lost", "left", nickname, false)
		if err != nil {
			log.Printf("Failed to create leave message for channel %d: %v", channelID, err)
			continue
		}

		// Broadcast leave event to other users in the channel
		leaveEvent := WSEvent{
			Type:      "event",
			ChannelID: channelID,
			Event:     "left",
			UserID:    userID,
			Nickname:  nickname,
			SentAt:    time.Now().UTC().Format(time.RFC3339),
		}

		h.sessions.BroadcastToChannel(channelID, leaveEvent)
	}

	// For logged-in users, disconnect but keep session alive for potential reconnection
	h.sessions.DisconnectSession(sessionID)
}

// generateJoinEventsForSessionRestore generates join events when a disconnected session reconnects
func (h *WebSocketHandler) generateJoinEventsForSessionRestore(session *chat.Session) {
	// Only generate events if user is logged in
	if session.UserID == nil || session.Nickname == nil {
		return
	}

	userID := *session.UserID
	nickname := *session.Nickname
	channels := session.GetChannels()

	if len(channels) == 0 {
		return
	}

	log.Printf("Generating join events for session restore of user %s (ID: %d) in %d channels", nickname, userID, len(channels))

	// Send join events to all channels the user is in
	for _, channelID := range channels {
		// For session restore, only broadcast to other users - don't create new DB records
		// since the user was already in the channel and there should be an existing join event
		joinEvent := WSEvent{
			Type:      "event",
			ChannelID: channelID,
			Event:     "joined",
			UserID:    userID,
			Nickname:  nickname,
			SentAt:    time.Now().UTC().Format(time.RFC3339),
		}
		h.sessions.BroadcastToChannel(channelID, joinEvent)
	}
}

// handleExpiredSession generates leave events when a session expires due to timeout
func (h *WebSocketHandler) handleExpiredSession(sessionID string) {
	session := h.sessions.GetSession(sessionID)
	if session == nil {
		return
	}

	// Check if user was logged in
	if session.UserID == nil || session.Nickname == nil {
		// Not logged in, just remove the session normally
		h.sessions.RemoveSession(sessionID)
		return
	}

	userID := *session.UserID
	nickname := *session.Nickname
	channels := session.GetChannels()

	log.Printf("Generating leave events for expired session of user %s (ID: %d)", nickname, userID)

	// Send leave events to all channels the user was in
	for _, channelID := range channels {
		// Create database record
		_, err := models.CreateMessage(h.db, &channelID, userID, "timed out", "left", nickname, false)
		if err != nil {
			log.Printf("Failed to create leave message for channel %d: %v", channelID, err)
			continue
		}

		// Broadcast leave event to other users in the channel
		leaveEvent := WSEvent{
			Type:      "event",
			ChannelID: channelID,
			Event:     "left",
			UserID:    userID,
			Nickname:  nickname,
			SentAt:    time.Now().UTC().Format(time.RFC3339),
		}

		h.sessions.BroadcastToChannel(channelID, leaveEvent)
	}

	// For logged-in users, disconnect but keep session alive for potential reconnection
	h.sessions.DisconnectSession(sessionID)
}
