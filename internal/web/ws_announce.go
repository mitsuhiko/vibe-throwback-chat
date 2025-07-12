package web

import (
	"log"
	"time"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type WSAnnounceRequest struct {
	WSRequest
	ChannelID *int   `json:"channel_id,omitempty"`
	Message   string `json:"message"`
}

type WSAnnounceResponse struct {
	ChannelID *int   `json:"channel_id,omitempty"`
	Message   string `json:"message"`
	Type      string `json:"type"`
}

func (h *WebSocketHandler) HandleAnnounce(sess *chat.Session, data []byte) error {
	var req WSAnnounceRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err
	}

	// Check if user is logged in
	if sess.UserID == nil {
		return sess.RespondError(req.ReqID, "Must be logged in to make announcements")
	}

	// Validate required fields
	if req.Message == "" {
		return sess.RespondError(req.ReqID, "Announcement message is required")
	}

	// Check if this is a channel announcement or server announcement
	if req.ChannelID != nil {
		// Channel announcement - check if user is operator
		isOp, err := models.IsUserOp(h.db, *sess.UserID, *req.ChannelID)
		if err != nil {
			log.Printf("Failed to check operator status: %v", err)
			return sess.RespondError(req.ReqID, "Database error")
		}
		if !isOp {
			return sess.RespondError(req.ReqID, "You must be an operator to make channel announcements")
		}

		// Verify the channel exists
		channel, err := models.GetChannelByID(h.db, *req.ChannelID)
		if err != nil {
			log.Printf("Failed to get channel by ID: %v", err)
			return sess.RespondError(req.ReqID, "Database error")
		}
		if channel == nil {
			return sess.RespondError(req.ReqID, "Channel not found")
		}

		// Create announcement event in database
		_, err = models.CreateMessage(h.db, req.ChannelID, *sess.UserID, req.Message, "announcement", *sess.Nickname, false)
		if err != nil {
			log.Printf("Failed to create announcement message: %v", err)
			return sess.RespondError(req.ReqID, "Failed to create announcement")
		}

		// Broadcast announcement event to all users in the channel
		announceEvent := WSEvent{
			Type:      "event",
			ChannelID: *req.ChannelID,
			Event:     "announcement",
			UserID:    *sess.UserID,
			Nickname:  *sess.Nickname,
			SentAt:    time.Now().Format(time.RFC3339),
		}
		h.sessions.BroadcastToChannel(*req.ChannelID, announceEvent)

		log.Printf("User %s made channel announcement in channel %s (ID: %d): %s",
			*sess.Nickname, channel.Name, channel.ID, req.Message)

		return sess.RespondSuccess(req.ReqID, WSAnnounceResponse{
			ChannelID: req.ChannelID,
			Message:   req.Message,
			Type:      "channel",
		})

	} else {
		// Server announcement - check if user is a service user (like ChanServ)
		user := &models.User{}
		err := h.db.ReadDBX().Get(user, "SELECT id, nickname, is_serv FROM users WHERE id = ?", *sess.UserID)
		if err != nil {
			log.Printf("Failed to get user: %v", err)
			return sess.RespondError(req.ReqID, "Database error")
		}

		if !user.IsServ {
			return sess.RespondError(req.ReqID, "Only service users can make server-wide announcements")
		}

		// Create server announcement event in database (no channel_id)
		_, err = models.CreateMessage(h.db, nil, *sess.UserID, req.Message, "announcement", *sess.Nickname, false)
		if err != nil {
			log.Printf("Failed to create server announcement message: %v", err)
			return sess.RespondError(req.ReqID, "Failed to create announcement")
		}

		// Broadcast announcement event to all connected users
		announceEvent := WSEvent{
			Type:      "event",
			ChannelID: 0, // 0 indicates server-wide announcement
			Event:     "announcement",
			UserID:    *sess.UserID,
			Nickname:  *sess.Nickname,
			SentAt:    time.Now().Format(time.RFC3339),
		}
		h.sessions.BroadcastToAll(announceEvent)

		log.Printf("User %s made server-wide announcement: %s", *sess.Nickname, req.Message)

		return sess.RespondSuccess(req.ReqID, WSAnnounceResponse{
			ChannelID: nil,
			Message:   req.Message,
			Type:      "server",
		})
	}
}
