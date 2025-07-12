package web

import (
	"log"
	"time"

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
		return sess.RespondError(req.ReqID, "Must be logged in to leave channels", nil)
	}

	var channel *models.Channel
	var err error

	// Find channel
	if req.ChannelName != "" {
		channel, err = models.GetChannelByName(h.db, req.ChannelName)
		if err != nil {
			return sess.RespondError(req.ReqID, "Database error", err)
		}
	} else if req.ChannelID != 0 {
		channel, err = models.GetChannelByID(h.db, req.ChannelID)
		if err != nil {
			return sess.RespondError(req.ReqID, "Database error", err)
		}
	} else {
		return sess.RespondError(req.ReqID, "Channel name or ID required", nil)
	}

	if channel == nil {
		return sess.RespondError(req.ReqID, "Channel not found", nil)
	}

	// Check if user is in the channel
	if !sess.IsInChannel(channel.ID) {
		return sess.RespondError(req.ReqID, "Not in channel", nil)
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
		SentAt:    time.Now().Format(time.RFC3339),
	}
	h.sessions.BroadcastToChannel(channel.ID, leaveEvent)

	// Remove operator status if user was an op
	err = models.RemoveUserOp(h.db, *sess.UserID, channel.ID)
	if err != nil {
		log.Printf("Failed to remove op status for user %d in channel %d: %v", *sess.UserID, channel.ID, err)
	}

	// Attempt to clean up the channel if it's now empty
	if err := models.DeleteEmptyChannel(h.db, channel.ID); err != nil {
		log.Printf("Failed to cleanup empty channel %d: %v", channel.ID, err)
	} else {
		log.Printf("Channel %s (ID: %d) was cleaned up as it's now empty", channel.Name, channel.ID)
	}

	log.Printf("User %s left channel %s (ID: %d)", *sess.Nickname, channel.Name, channel.ID)

	return sess.RespondSuccess(req.ReqID, WSLeaveResponse{
		ChannelID:   channel.ID,
		ChannelName: channel.Name,
	})
}
