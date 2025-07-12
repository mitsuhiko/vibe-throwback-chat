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
		return sess.RespondError(req.ReqID, "Must be logged in to list channels", nil)
	}

	// Get all channels from database
	var dbChannels []models.Channel
	err := h.db.ReadDBX().Select(&dbChannels, "SELECT id, name, topic FROM channels ORDER BY name")
	if err != nil {
		log.Printf("Failed to get channels: %v", err)
		return sess.RespondError(req.ReqID, "Failed to retrieve channel list", nil)
	}

	// Build channel info with session-based user counts
	var channels []models.ChannelInfo
	for _, channel := range dbChannels {
		// Get current user count from session state (not database reconstruction)
		userCount := h.sessions.GetChannelUserCount(channel.ID)

		channels = append(channels, models.ChannelInfo{
			ID:        channel.ID,
			Name:      channel.Name,
			Topic:     channel.Topic,
			UserCount: userCount,
		})
	}

	log.Printf("User %s requested channel list, returning %d channels with session-based user counts", *sess.Nickname, len(channels))

	return sess.RespondSuccess(req.ReqID, WSListChannelsResponse{
		Channels: channels,
	})
}
