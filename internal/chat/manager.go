package chat

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

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
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	// Start heartbeat checker
	go sm.heartbeatChecker()

	return sm
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

func (sm *SessionManager) heartbeatChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		sm.cleanupExpiredSessions()
	}
}

func (sm *SessionManager) cleanupExpiredSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute) // 5 missed heartbeats (60s each)

	for sessionID, session := range sm.sessions {
		session.mu.Lock()
		lastHeartbeat := session.LastHeartbeat
		session.mu.Unlock()

		if lastHeartbeat.Before(cutoff) {
			log.Printf("Session %s expired (last heartbeat: %v)", sessionID, lastHeartbeat)
			session.Conn.Close()
			delete(sm.sessions, sessionID)
		}
	}
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

	return s.Conn.WriteJSON(message)
}

// RespondError sends an error response for a WebSocket request
func (s *Session) RespondError(reqID string, errorMsg string) error {
	response := map[string]interface{}{
		"type":   "response",
		"req_id": reqID,
		"okay":   false,
		"error":  errorMsg,
	}
	return s.SendMessage(response)
}

// RespondSuccess sends a success response for a WebSocket request
func (s *Session) RespondSuccess(reqID string, data interface{}) error {
	response := map[string]interface{}{
		"type":   "response",
		"req_id": reqID,
		"okay":   true,
		"data":   data,
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
