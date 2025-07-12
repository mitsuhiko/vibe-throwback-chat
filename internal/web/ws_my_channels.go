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

	// Get channels from current session state (not database reconstruction)
	channelIDs := sess.GetChannels()
	var channels []models.ChannelInfo

	for _, channelID := range channelIDs {
		// Get channel metadata from database
		channel, err := models.GetChannelByID(h.db, channelID)
		if err != nil {
			log.Printf("Failed to get channel %d metadata: %v", channelID, err)
			continue // Skip this channel if we can't get its metadata
		}
		if channel == nil {
			log.Printf("Channel %d not found in database", channelID)
			continue // Skip this channel if it doesn't exist
		}

		// Get current user count from session state (not database reconstruction)
		userCount := h.sessions.GetChannelUserCount(channelID)

		// Build ChannelInfo
		channels = append(channels, models.ChannelInfo{
			ID:        channel.ID,
			Name:      channel.Name,
			Topic:     channel.Topic,
			UserCount: userCount,
		})
	}

	log.Printf("User %s requested their channel list, returning %d channels from session state", *sess.Nickname, len(channels))

	return sess.RespondSuccess(req.ReqID, WSMyChannelsResponse{
		Channels: channels,
	})
}
