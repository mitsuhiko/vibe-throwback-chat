package web

import (
	"log"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type WSMyChannelsRequest struct {
	WSRequest
}

type WSMyChannelsResponse struct {
	Channels []models.ChannelInfo `json:"channels"`
}

func (h *WebSocketHandler) HandleMyChannels(sess *chat.Session, data []byte) error {
	var req WSMyChannelsRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err
	}

	// Check if user is logged in
	if sess.UserID == nil {
		return sess.RespondError(req.ReqID, "Must be logged in to list your channels", nil)
	}

	// Get channels the user is currently in
	channels, err := models.GetUserChannels(h.db, *sess.UserID)
	if err != nil {
		log.Printf("Failed to get user channels for user %d: %v", *sess.UserID, err)
		return sess.RespondError(req.ReqID, "Failed to retrieve your channel list", nil)
	}

	log.Printf("User %s requested their channel list, returning %d channels", *sess.Nickname, len(channels))

	return sess.RespondSuccess(req.ReqID, WSMyChannelsResponse{
		Channels: channels,
	})
}
