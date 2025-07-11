package web

import (
	"log"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type WSJoinRequest struct {
	WSRequest
	ChannelName string `json:"channel_name,omitempty"`
	ChannelID   int    `json:"channel_id,omitempty"`
}

type WSJoinResponse struct {
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
}

type WSEvent struct {
	Type      string `json:"type"`
	ChannelID int    `json:"channel_id"`
	Event     string `json:"event"`
	UserID    int    `json:"user_id"`
	Nickname  string `json:"nickname"`
	SentAt    string `json:"sent_at"`
}

func (h *WebSocketHandler) HandleJoin(sess *chat.Session, data []byte) error {
	var req WSJoinRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err
	}

	// Check if user is logged in
	if sess.UserID == nil {
		return sess.SendMessage(NewWSResponse(req.ReqID, false, "Must be logged in to join channels", nil))
	}

	var channel *models.Channel
	var err error

	// Find or create channel
	if req.ChannelName != "" {
		channel, err = models.GetChannelByName(h.db, req.ChannelName)
		if err != nil {
			log.Printf("Failed to get channel by name: %v", err)
			return sess.SendMessage(NewWSResponse(req.ReqID, false, "Database error", nil))
		}
		if channel == nil {
			// Create new channel
			channel, err = models.CreateChannel(h.db, req.ChannelName)
			if err != nil {
				log.Printf("Failed to create channel: %v", err)
				return sess.SendMessage(NewWSResponse(req.ReqID, false, "Failed to create channel", nil))
			}
		}
	} else if req.ChannelID != 0 {
		channel, err = models.GetChannelByID(h.db, req.ChannelID)
		if err != nil {
			log.Printf("Failed to get channel by ID: %v", err)
			return sess.SendMessage(NewWSResponse(req.ReqID, false, "Database error", nil))
		}
		if channel == nil {
			return sess.SendMessage(NewWSResponse(req.ReqID, false, "Channel not found", nil))
		}
	} else {
		return sess.SendMessage(NewWSResponse(req.ReqID, false, "Channel name or ID required", nil))
	}

	// Check if user is already in the channel
	if sess.IsInChannel(channel.ID) {
		return sess.SendMessage(NewWSResponse(req.ReqID, false, "Already in channel", nil))
	}

	// Add user to channel subscription
	sess.JoinChannel(channel.ID)

	// Check if channel is empty and make user op if so
	isEmpty, err := models.IsChannelEmpty(h.db, channel.ID)
	if err != nil {
		log.Printf("Failed to check if channel is empty: %v", err)
	} else if isEmpty {
		err = models.MakeUserOp(h.db, *sess.UserID, channel.ID, 1) // ChanServ grants op
		if err != nil {
			log.Printf("Failed to make user op: %v", err)
		}
	}

	// Create join event in database
	_, err = models.CreateMessage(h.db, &channel.ID, *sess.UserID, "", "joined", *sess.Nickname, false)
	if err != nil {
		log.Printf("Failed to create join message: %v", err)
	}

	// Broadcast join event to all users in the channel
	joinEvent := WSEvent{
		Type:      "event",
		ChannelID: channel.ID,
		Event:     "joined",
		UserID:    *sess.UserID,
		Nickname:  *sess.Nickname,
		SentAt:    "",
	}
	h.sessions.BroadcastToChannel(channel.ID, joinEvent)

	log.Printf("User %s joined channel %s (ID: %d)", *sess.Nickname, channel.Name, channel.ID)

	return sess.SendMessage(NewWSResponse(req.ReqID, true, "", WSJoinResponse{
		ChannelID:   channel.ID,
		ChannelName: channel.Name,
	}))
}
