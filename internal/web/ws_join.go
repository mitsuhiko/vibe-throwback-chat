package web

import (
	"log"
	"time"

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
		return sess.RespondError(req.ReqID, "Must be logged in to join channels", nil)
	}

	var channel *models.Channel
	var err error

	// Find or create channel
	if req.ChannelName != "" {
		channel, err = models.GetChannelByName(h.db, req.ChannelName)
		if err != nil {
			return sess.RespondError(req.ReqID, "Database error", err)
		}
		if channel == nil {
			// Create new channel
			channel, err = models.CreateChannel(h.db, req.ChannelName)
			if err != nil {
				return sess.RespondError(req.ReqID, "Failed to create channel", err)
			}
		}
	} else if req.ChannelID != 0 {
		channel, err = models.GetChannelByID(h.db, req.ChannelID)
		if err != nil {
			return sess.RespondError(req.ReqID, "Database error", err)
		}
		if channel == nil {
			return sess.RespondError(req.ReqID, "Channel not found", nil)
		}
	} else {
		return sess.RespondError(req.ReqID, "Channel name or ID required", nil)
	}

	// Check if user is already in the channel
	if sess.IsInChannel(channel.ID) {
		return sess.RespondError(req.ReqID, "Already in channel", nil)
	}

	// Add user to channel subscription
	sess.JoinChannel(channel.ID)

	// Check if channel is empty and make user op if so
	isEmpty, err := models.IsChannelEmpty(h.db, channel.ID)
	if err != nil {
		log.Printf("Failed to check if channel %d is empty: %v", channel.ID, err)
	} else if isEmpty {
		err = models.MakeUserOp(h.db, *sess.UserID, channel.ID, 1) // ChanServ grants op
		if err != nil {
			log.Printf("Failed to make user %d op in channel %d: %v", *sess.UserID, channel.ID, err)
		}
	}

	// Create join event in database
	_, err = models.CreateMessage(h.db, &channel.ID, *sess.UserID, "", "joined", *sess.Nickname, false)
	if err != nil {
		log.Printf("Failed to create join message for user %d in channel %d: %v", *sess.UserID, channel.ID, err)
	}

	// Broadcast join event to all users in the channel
	joinEvent := WSEvent{
		Type:      "event",
		ChannelID: channel.ID,
		Event:     "joined",
		UserID:    *sess.UserID,
		Nickname:  *sess.Nickname,
		SentAt:    time.Now().Format(time.RFC3339),
	}
	h.sessions.BroadcastToChannel(channel.ID, joinEvent)

	// Send initial room content (last 100 messages and events)
	historyOptions := models.MessageHistoryOptions{
		Limit: 100,
	}
	recentMessages, err := models.GetMessageHistory(h.db, channel.ID, historyOptions)
	if err != nil {
		log.Printf("Failed to fetch recent messages for channel %d: %v", channel.ID, err)
	} else {
		// Send messages in chronological order (reverse the DESC order from DB)
		for i := len(recentMessages) - 1; i >= 0; i-- {
			msg := recentMessages[i]

			if msg.Event != "" && msg.Event != "message" {
				// Send as event
				eventMsg := WSEvent{
					Type:      "event",
					ChannelID: channel.ID,
					Event:     msg.Event,
					UserID:    msg.UserID,
					Nickname:  msg.Nickname,
					SentAt:    msg.SentAt.Format(time.RFC3339),
				}
				sess.SendMessage(eventMsg)
			} else {
				// Send as regular message
				chatMsg := WSMessage{
					Type:      "message",
					ChannelID: channel.ID,
					Message:   msg.Message,
					IsPassive: msg.IsPassive,
					SentAt:    msg.SentAt.Format(time.RFC3339),
					UserID:    msg.UserID,
					Nickname:  msg.Nickname,
				}
				sess.SendMessage(chatMsg)
			}
		}
	}

	log.Printf("User %s joined channel %s (ID: %d)", *sess.Nickname, channel.Name, channel.ID)

	return sess.RespondSuccess(req.ReqID, WSJoinResponse{
		ChannelID:   channel.ID,
		ChannelName: channel.Name,
	})
}
