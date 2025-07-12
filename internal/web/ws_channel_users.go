package web

import (
	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type ChannelUsersRequest struct {
	Cmd       string `json:"cmd"`
	ChannelID int    `json:"channel_id"`
	ReqID     string `json:"req_id"`
}

func (h *WebSocketHandler) HandleChannelUsers(sess *chat.Session, data []byte) error {
	var req ChannelUsersRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err
	}

	// Ensure user is logged in
	if sess.UserID == nil {
		return sess.RespondError(req.ReqID, "Not logged in", nil)
	}

	// Verify channel exists
	channel, err := models.GetChannelByID(h.db, req.ChannelID)
	if err != nil {
		return sess.RespondError(req.ReqID, "Database error", err)
	}
	if channel == nil {
		return sess.RespondError(req.ReqID, "Channel not found", nil)
	}

	// Verify user is in the channel
	if !sess.IsInChannel(req.ChannelID) {
		return sess.RespondError(req.ReqID, "Not in channel", nil)
	}

	// Get channel users
	users, err := models.GetChannelUsers(h.db, req.ChannelID)
	if err != nil {
		return sess.RespondError(req.ReqID, "Failed to get channel users", err)
	}

	// Send response
	response := map[string]interface{}{
		"users": users,
	}

	return sess.RespondSuccess(req.ReqID, response)
}