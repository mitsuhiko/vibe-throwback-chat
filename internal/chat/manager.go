package chat

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSResponse represents a WebSocket response message
type WSResponse struct {
	Type  string      `json:"type"`
	ReqID string      `json:"req_id"`
	Okay  bool        `json:"okay"`
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

type Session struct {
	ID            string          `json:"id"`
	UserID        *int            `json:"user_id,omitempty"`
	Nickname      *string         `json:"nickname,omitempty"`
	Conn          *websocket.Conn `json:"-"`
	LastHeartbeat time.Time       `json:"last_heartbeat"`
	Channels      map[int]bool    `json:"channels"` // channel IDs user is subscribed to
	mu            sync.Mutex      `json:"-"`
}

type SessionManager struct {
	sessions         map[string]*Session
	mu               sync.RWMutex
	onSessionExpired func(sessionID string) // callback for handling expired sessions
}

func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	// Start heartbeat checker
	go sm.heartbeatChecker()

	return sm
}

// SetSessionExpiredCallback sets the callback function for handling expired sessions
func (sm *SessionManager) SetSessionExpiredCallback(callback func(sessionID string)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onSessionExpired = callback
}

func (sm *SessionManager) AddSession(sessionID string, conn *websocket.Conn) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &Session{
		ID:            sessionID,
		Conn:          conn,
		LastHeartbeat: time.Now(),
		Channels:      make(map[int]bool),
	}

	sm.sessions[sessionID] = session
	log.Printf("Session %s added", sessionID)

	return session
}

func (sm *SessionManager) GetSession(sessionID string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.sessions[sessionID]
}

func (sm *SessionManager) RemoveSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.Conn.Close()
		delete(sm.sessions, sessionID)
		log.Printf("Session %s removed", sessionID)
	}
}

// DisconnectSession closes the WebSocket connection but keeps the session alive for reconnection
func (sm *SessionManager) DisconnectSession(sessionID string) {
	sm.mu.RLock()
	session := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if session != nil {
		session.mu.Lock()
		if session.Conn != nil {
			session.Conn.Close()
			session.Conn = nil // Clear the connection but keep the session
		}
		session.mu.Unlock()
		log.Printf("Session %s disconnected but kept alive for reconnection", sessionID)
	}
}

// TransferConnection updates an existing session with a new WebSocket connection
func (sm *SessionManager) TransferConnection(sessionID string, conn *websocket.Conn) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.mu.Lock()
		// Close the old connection if it exists
		if session.Conn != nil {
			session.Conn.Close()
		}
		// Assign the new connection
		session.Conn = conn
		session.LastHeartbeat = time.Now()
		session.mu.Unlock()
		log.Printf("Connection transferred to session %s", sessionID)
	}
}

func (sm *SessionManager) UpdateHeartbeat(sessionID string) {
	sm.mu.RLock()
	session := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if session != nil {
		session.mu.Lock()
		session.LastHeartbeat = time.Now()
		session.mu.Unlock()
	}
}

func (sm *SessionManager) GetSessions() map[string]*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Return a copy to avoid race conditions
	sessions := make(map[string]*Session)
	for k, v := range sm.sessions {
		sessions[k] = v
	}
	return sessions
}

func (sm *SessionManager) GetSessionsByUserID(userID int) []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var userSessions []*Session
	for _, session := range sm.sessions {
		if session.UserID != nil && *session.UserID == userID {
			userSessions = append(userSessions, session)
		}
	}
	return userSessions
}

func (sm *SessionManager) BroadcastToChannel(channelID int, message interface{}) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, session := range sm.sessions {
		if session.IsInChannel(channelID) {
			go func(s *Session) {
				if err := s.SendMessage(message); err != nil {
					log.Printf("Failed to send message to session %s: %v", s.ID, err)
				}
			}(session)
		}
	}
}

func (sm *SessionManager) BroadcastToAll(message interface{}) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, session := range sm.sessions {
		// Only send to logged in users
		if session.UserID != nil {
			go func(s *Session) {
				if err := s.SendMessage(message); err != nil {
					log.Printf("Failed to send message to session %s: %v", s.ID, err)
				}
			}(session)
		}
	}
}

// GetChannelUserCount returns the number of active sessions in a channel
func (sm *SessionManager) GetChannelUserCount(channelID int) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	count := 0
	for _, session := range sm.sessions {
		// Only count logged in users who are in the channel
		if session.UserID != nil && session.IsInChannel(channelID) {
			count++
		}
	}
	return count
}

func (sm *SessionManager) heartbeatChecker() {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		sm.cleanupExpiredSessions()
	}
}

func (sm *SessionManager) cleanupExpiredSessions() {
	cutoff := time.Now().Add(-60 * time.Second) // timeout after 60 seconds
	var expiredSessions []string

	// First pass: identify expired sessions
	sm.mu.RLock()
	for sessionID, session := range sm.sessions {
		session.mu.Lock()
		lastHeartbeat := session.LastHeartbeat
		hasActiveConnection := session.Conn != nil
		session.mu.Unlock()

		if lastHeartbeat.Before(cutoff) && hasActiveConnection {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}
	sm.mu.RUnlock()

	// Second pass: handle expired sessions
	for _, sessionID := range expiredSessions {
		session := sm.GetSession(sessionID)
		if session == nil {
			continue
		}

		log.Printf("Session %s expired (last heartbeat: %v)", sessionID, session.LastHeartbeat)

		// Check if user was logged in
		if session.UserID != nil && session.Nickname != nil {
			// For logged-in users: call callback to generate leave events
			if sm.onSessionExpired != nil {
				sm.onSessionExpired(sessionID)
			} else {
				// Fallback: just disconnect but keep session alive
				sm.DisconnectSession(sessionID)
			}
		} else {
			// Not logged in, remove completely
			sm.RemoveSession(sessionID)
		}
	}
}

// RemoveSessionWithLeaveEvents removes a session and returns info needed to generate leave events
func (sm *SessionManager) RemoveSessionWithLeaveEvents(sessionID string) (userID *int, nickname *string, channels []int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.mu.Lock()
		userID = session.UserID
		nickname = session.Nickname
		// Get channels manually to avoid calling a method that might need the lock
		channels = make([]int, 0, len(session.Channels))
		for channelID := range session.Channels {
			channels = append(channels, channelID)
		}
		session.mu.Unlock()

		session.Conn.Close()
		delete(sm.sessions, sessionID)
		log.Printf("Session %s removed with leave events", sessionID)
	}
	return
}

func (s *Session) SetUser(userID int, nickname string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.UserID = &userID
	s.Nickname = &nickname
}

func (s *Session) ClearUser() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.UserID = nil
	s.Nickname = nil
}

func (s *Session) SendMessage(message interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Conn == nil {
		return fmt.Errorf("session %s has no active connection", s.ID)
	}

	return s.Conn.WriteJSON(message)
}

// RespondError sends an error response for a WebSocket request
// If originalErr is provided, it will be logged with the error message
func (s *Session) RespondError(reqID string, errorMsg string, originalErr error) error {
	// Log the original error if provided
	if originalErr != nil {
		log.Printf("Session %s error (req_id: %s): %s - original error: %v", s.ID, reqID, errorMsg, originalErr)
	}

	response := WSResponse{
		Type:  "response",
		ReqID: reqID,
		Okay:  false,
		Error: errorMsg,
	}
	return s.SendMessage(response)
}

// RespondSuccess sends a success response for a WebSocket request
func (s *Session) RespondSuccess(reqID string, data interface{}) error {
	response := WSResponse{
		Type:  "response",
		ReqID: reqID,
		Okay:  true,
		Data:  data,
	}
	return s.SendMessage(response)
}

func (s *Session) JoinChannel(channelID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Channels[channelID] = true
}

func (s *Session) LeaveChannel(channelID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Channels, channelID)
}

func (s *Session) IsInChannel(channelID int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Channels[channelID]
}

func (s *Session) GetChannels() []int {
	s.mu.Lock()
	defer s.mu.Unlock()

	channels := make([]int, 0, len(s.Channels))
	for channelID := range s.Channels {
		channels = append(channels, channelID)
	}
	return channels
}
