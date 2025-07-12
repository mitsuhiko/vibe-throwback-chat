package web

import (
	"log"
	"time"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type WSNickRequest struct {
	WSRequest
	NewNickname string `json:"new_nickname"`
}

type WSNickResponse struct {
	UserID      int    `json:"user_id"`
	OldNickname string `json:"old_nickname"`
	NewNickname string `json:"new_nickname"`
}

func (h *WebSocketHandler) HandleNick(sess *chat.Session, data []byte) error {
	var req WSNickRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err // This will terminate the websocket connection
	}

	// Check if user is logged in
	if sess.UserID == nil {
		return sess.RespondError(req.ReqID, "Must be logged in to change nickname")
	}

	// Validate new nickname
	if req.NewNickname == "" {
		return sess.RespondError(req.ReqID, "New nickname is required")
	}

	// Get current nickname
	oldNickname := *sess.Nickname

	// Check if the new nickname is the same as current
	if req.NewNickname == oldNickname {
		return sess.RespondError(req.ReqID, "New nickname must be different from current nickname")
	}

	// Check if new nickname is already taken by another active session
	for _, s := range h.sessions.GetSessions() {
		if s.Nickname != nil && *s.Nickname == req.NewNickname && s.ID != sess.ID {
			return sess.RespondError(req.ReqID, "Nickname already in use")
		}
	}

	// Update nickname in database
	if err := models.UpdateUserNickname(h.db, *sess.UserID, req.NewNickname); err != nil {
		log.Printf("Failed to update user nickname: %v", err)
		return sess.RespondError(req.ReqID, "Database error")
	}

	// Update session nickname
	sess.SetUser(*sess.UserID, req.NewNickname)

	// Get all channels the user is in to broadcast nick change event
	userChannels := sess.GetChannels()

	// Create nick change events in database and broadcast to all channels user is in
	for _, channelID := range userChannels {
		// Create nick change event in database
		_, err := models.CreateMessage(h.db, &channelID, *sess.UserID, "", "nick_change", req.NewNickname, false)
		if err != nil {
			log.Printf("Failed to create nick change message for channel %d: %v", channelID, err)
			// Continue to other channels even if one fails
			continue
		}

		// Broadcast nick change event to all users in the channel
		nickChangeEvent := WSEvent{
			Type:      "event",
			ChannelID: channelID,
			Event:     "nick_change",
			UserID:    *sess.UserID,
			Nickname:  req.NewNickname,
			SentAt:    time.Now().Format(time.RFC3339),
		}
		h.sessions.BroadcastToChannel(channelID, nickChangeEvent)
	}

	log.Printf("User %s (ID: %d) changed nickname to %s on session %s", oldNickname, *sess.UserID, req.NewNickname, sess.ID)

	return sess.RespondSuccess(req.ReqID, WSNickResponse{
		UserID:      *sess.UserID,
		OldNickname: oldNickname,
		NewNickname: req.NewNickname,
	})
}
