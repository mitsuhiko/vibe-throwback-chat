package web

import (
	"log"
	"time"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type WSMeRequest struct {
	WSRequest
	ChannelID int    `json:"channel_id"`
	Message   string `json:"message"`
}

func (h *WebSocketHandler) HandleMe(sess *chat.Session, data []byte) error {
	var req WSMeRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err
	}

	// Check if user is logged in
	if sess.UserID == nil {
		return sess.RespondError(req.ReqID, "Must be logged in to send messages", nil)
	}

	// Check if message is not empty
	if req.Message == "" {
		return sess.RespondError(req.ReqID, "Message cannot be empty", nil)
	}

	// Check if channel exists
	channel, err := models.GetChannelByID(h.db, req.ChannelID)
	if err != nil {
		return sess.RespondError(req.ReqID, "Database error", err)
	}
	if channel == nil {
		return sess.RespondError(req.ReqID, "Channel not found", nil)
	}

	// Check if user is in the channel
	if !sess.IsInChannel(req.ChannelID) {
		return sess.RespondError(req.ReqID, "Not in channel", nil)
	}

	// Create passive message in database (is_passive = true for /me commands)
	dbMessage, err := models.CreateMessage(h.db, &req.ChannelID, *sess.UserID, req.Message, "message", *sess.Nickname, true)
	if err != nil {
		return sess.RespondError(req.ReqID, "Failed to send message", err)
	}

	// Broadcast passive message to all users in the channel
	wsMessage := WSMessage{
		Type:      "message",
		ChannelID: req.ChannelID,
		Message:   req.Message,
		IsPassive: true, // Always true for /me commands
		SentAt:    dbMessage.SentAt.Format(time.RFC3339),
		UserID:    *sess.UserID,
		Nickname:  *sess.Nickname,
	}
	h.sessions.BroadcastToChannel(req.ChannelID, wsMessage)

	log.Printf("Me message sent by %s to channel %d: %s", *sess.Nickname, req.ChannelID, req.Message)

	return sess.RespondSuccess(req.ReqID, nil)
}
