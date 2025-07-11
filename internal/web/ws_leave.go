package web

import (
	"log"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type WSLeaveRequest struct {
	WSRequest
	ChannelName string `json:"channel_name,omitempty"`
	ChannelID   int    `json:"channel_id,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type WSLeaveResponse struct {
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
}

func (h *WebSocketHandler) HandleLeave(sess *chat.Session, data []byte) error {
	var req WSLeaveRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err
	}

	// Check if user is logged in
	if sess.UserID == nil {
		return sess.RespondError(req.ReqID, "Must be logged in to leave channels")
	}

	var channel *models.Channel
	var err error

	// Find channel
	if req.ChannelName != "" {
		channel, err = models.GetChannelByName(h.db, req.ChannelName)
		if err != nil {
			log.Printf("Failed to get channel by name: %v", err)
			return sess.RespondError(req.ReqID, "Database error")
		}
	} else if req.ChannelID != 0 {
		channel, err = models.GetChannelByID(h.db, req.ChannelID)
		if err != nil {
			log.Printf("Failed to get channel by ID: %v", err)
			return sess.RespondError(req.ReqID, "Database error")
		}
	} else {
		return sess.RespondError(req.ReqID, "Channel name or ID required")
	}

	if channel == nil {
		return sess.RespondError(req.ReqID, "Channel not found")
	}

	// Check if user is in the channel
	if !sess.IsInChannel(channel.ID) {
		return sess.RespondError(req.ReqID, "Not in channel")
	}

	// Remove user from channel subscription
	sess.LeaveChannel(channel.ID)

	// Create leave event message
	leaveMessage := ""
	if req.Reason != "" {
		leaveMessage = req.Reason
	}

	// Create leave event in database
	_, err = models.CreateMessage(h.db, &channel.ID, *sess.UserID, leaveMessage, "left", *sess.Nickname, false)
	if err != nil {
		log.Printf("Failed to create leave message: %v", err)
	}

	// Broadcast leave event to all users in the channel
	leaveEvent := WSEvent{
		Type:      "event",
		ChannelID: channel.ID,
		Event:     "left",
		UserID:    *sess.UserID,
		Nickname:  *sess.Nickname,
		SentAt:    "",
	}
	h.sessions.BroadcastToChannel(channel.ID, leaveEvent)

	log.Printf("User %s left channel %s (ID: %d)", *sess.Nickname, channel.Name, channel.ID)

	return sess.RespondSuccess(req.ReqID, WSLeaveResponse{
		ChannelID:   channel.ID,
		ChannelName: channel.Name,
	})
}
