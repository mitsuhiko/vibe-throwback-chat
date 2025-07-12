package web

import (
	"log"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type WSListChannelsRequest struct {
	WSRequest
}

type WSListChannelsResponse struct {
	Channels []models.ChannelInfo `json:"channels"`
}

func (h *WebSocketHandler) HandleListChannels(sess *chat.Session, data []byte) error {
	var req WSListChannelsRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err
	}

	// Check if user is logged in
	if sess.UserID == nil {
		return sess.RespondError(req.ReqID, "Must be logged in to list channels")
	}

	// Get all channels with their user counts
	channels, err := models.GetAllChannelsWithInfo(h.db)
	if err != nil {
		log.Printf("Failed to get channels with info: %v", err)
		return sess.RespondError(req.ReqID, "Failed to retrieve channel list")
	}

	log.Printf("User %s requested channel list, returning %d channels", *sess.Nickname, len(channels))

	return sess.RespondSuccess(req.ReqID, WSListChannelsResponse{
		Channels: channels,
	})
}
