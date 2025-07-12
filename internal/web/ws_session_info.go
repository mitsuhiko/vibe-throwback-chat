package web

import (
	"log"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type SessionInfoRequest struct {
	Cmd   string `json:"cmd"`
	ReqID string `json:"req_id"`
}

func (h *WebSocketHandler) HandleSessionInfo(sess *chat.Session, data []byte) error {
	var req SessionInfoRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err
	}

	// Check if user is logged in
	if sess.UserID == nil || sess.Nickname == nil {
		return sess.RespondError(req.ReqID, "Not logged in", nil)
	}

	// Get user's channels
	userChannels := sess.GetChannels()

	// Convert to channel info format
	channels := make([]map[string]interface{}, 0, len(userChannels))
	for _, channelID := range userChannels {
		// Get channel info from database
		if channel := h.getChannelInfo(channelID); channel != nil {
			channels = append(channels, map[string]interface{}{
				"id":    channelID,
				"name":  channel["name"],
				"topic": channel["topic"],
			})
		}
	}

	responseData := map[string]interface{}{
		"session_id": sess.ID,
		"user_id":    *sess.UserID,
		"nickname":   *sess.Nickname,
		"channels":   channels,
	}

	return sess.RespondSuccess(req.ReqID, responseData)
}

// Helper function to get channel info from database
func (h *WebSocketHandler) getChannelInfo(channelID int) map[string]interface{} {
	channel, err := models.GetChannelByID(h.db, channelID)
	if err != nil {
		log.Printf("Failed to get channel info for ID %d: %v", channelID, err)
		return nil
	}
	if channel == nil {
		log.Printf("Channel with ID %d not found", channelID)
		return nil
	}

	return map[string]interface{}{
		"name":  channel.Name,
		"topic": channel.Topic,
	}
}
