package web

import (
	"log"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type WSLoginRequest struct {
	WSRequest
	Nickname string `json:"nickname"`
}

type WSLoginResponse struct {
	UserID   int    `json:"user_id"`
	Nickname string `json:"nickname"`
}

func (h *WebSocketHandler) HandleLogin(sess *chat.Session, data []byte) error {
	var req WSLoginRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err // This will terminate the websocket connection
	}

	if req.Nickname == "" {
		return sess.SendMessage(NewWSResponse(req.ReqID, false, "Nickname is required", nil))
	}

	// Check if user is already logged in
	if sess.UserID != nil {
		return sess.SendMessage(NewWSResponse(req.ReqID, false, "Already logged in", nil))
	}

	// Check if nickname is already taken by another active session
	for _, s := range h.sessions.GetSessions() {
		if s.Nickname != nil && *s.Nickname == req.Nickname && s.ID != sess.ID {
			return sess.SendMessage(NewWSResponse(req.ReqID, false, "Nickname already in use", nil))
		}
	}

	// Create or get user from database
	user, err := models.CreateOrUpdateUser(h.db, req.Nickname)
	if err != nil {
		log.Printf("Failed to create/update user: %v", err)
		return sess.SendMessage(NewWSResponse(req.ReqID, false, "Database error", nil))
	}

	// Set user in session
	sess.SetUser(user.ID, user.Nickname)

	log.Printf("User %s (ID: %d) logged in on session %s", user.Nickname, user.ID, sess.ID)

	return sess.SendMessage(NewWSResponse(req.ReqID, true, "", WSLoginResponse{
		UserID:   user.ID,
		Nickname: user.Nickname,
	}))
}
