package web

import (
	"log"
	"time"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type WSMessageRequest struct {
	WSRequest
	ChannelID int    `json:"channel_id"`
	Message   string `json:"message"`
	IsPassive bool   `json:"is_passive"`
}

type WSMessage struct {
	Type      string `json:"type"`
	ChannelID int    `json:"channel_id"`
	Message   string `json:"message"`
	IsPassive bool   `json:"is_passive"`
	SentAt    string `json:"sent_at"`
	UserID    int    `json:"user_id"`
	Nickname  string `json:"nickname"`
}

func (h *WebSocketHandler) HandleMessage(sess *chat.Session, data []byte) error {
	var req WSMessageRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err
	}

	// Check if user is logged in
	if sess.UserID == nil {
		return sess.SendMessage(NewWSResponse(req.ReqID, false, "Must be logged in to send messages", nil))
	}

	// Check if message is not empty
	if req.Message == "" {
		return sess.SendMessage(NewWSResponse(req.ReqID, false, "Message cannot be empty", nil))
	}

	// Check if channel exists
	channel, err := models.GetChannelByID(h.db, req.ChannelID)
	if err != nil {
		log.Printf("Failed to get channel: %v", err)
		return sess.SendMessage(NewWSResponse(req.ReqID, false, "Database error", nil))
	}
	if channel == nil {
		return sess.SendMessage(NewWSResponse(req.ReqID, false, "Channel not found", nil))
	}

	// Check if user is in the channel
	if !sess.IsInChannel(req.ChannelID) {
		return sess.SendMessage(NewWSResponse(req.ReqID, false, "Not in channel", nil))
	}

	// Create message in database
	dbMessage, err := models.CreateMessage(h.db, &req.ChannelID, *sess.UserID, req.Message, "message", *sess.Nickname, req.IsPassive)
	if err != nil {
		log.Printf("Failed to create message: %v", err)
		return sess.SendMessage(NewWSResponse(req.ReqID, false, "Failed to send message", nil))
	}

	// Broadcast message to all users in the channel
	wsMessage := WSMessage{
		Type:      "message",
		ChannelID: req.ChannelID,
		Message:   req.Message,
		IsPassive: req.IsPassive,
		SentAt:    dbMessage.SentAt.Format(time.RFC3339),
		UserID:    *sess.UserID,
		Nickname:  *sess.Nickname,
	}
	h.sessions.BroadcastToChannel(req.ChannelID, wsMessage)

	log.Printf("Message sent by %s to channel %d: %s", *sess.Nickname, req.ChannelID, req.Message)

	return sess.SendMessage(NewWSResponse(req.ReqID, true, "", nil))
}