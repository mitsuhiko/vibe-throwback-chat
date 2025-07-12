package web

import (
	"log"
	"time"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type QuitRequest struct {
	Cmd          string  `json:"cmd"`
	ReqID        string  `json:"req_id"`
	DyingMessage *string `json:"dying_message,omitempty"`
}

func (h *WebSocketHandler) HandleQuit(sess *chat.Session, data []byte) error {
	var req QuitRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err
	}

	// Only allow logged in users to quit
	if sess.UserID == nil {
		return sess.RespondError(req.ReqID, "Not logged in", nil)
	}

	userID := *sess.UserID
	nickname := *sess.Nickname
	dyingMessage := "has quit"
	if req.DyingMessage != nil && *req.DyingMessage != "" {
		dyingMessage = *req.DyingMessage
	}

	// Get all channels the user is in before we remove the session
	userChannels := sess.GetChannels()

	// Send leave events to all channels
	for _, channelID := range userChannels {
		// Create database record
		_, err := models.CreateMessage(h.db, &channelID, userID, dyingMessage, "left", nickname, false)
		if err != nil {
			// Log error but continue with other channels
			log.Printf("Failed to create leave message for channel %d: %v", channelID, err)
		}

		// Broadcast leave event to other users in the channel
		leaveEvent := WSEvent{
			Type:      "event",
			ChannelID: channelID,
			Event:     "left",
			UserID:    userID,
			Nickname:  nickname,
			SentAt:    time.Now().UTC().Format(time.RFC3339),
		}

		h.sessions.BroadcastToChannel(channelID, leaveEvent)

		// Remove user from channel subscription
		sess.LeaveChannel(channelID)
	}

	// Send success response
	if err := sess.RespondSuccess(req.ReqID, QuitResponse{Message: "Goodbye!"}); err != nil {
		log.Printf("Failed to send quit response: %v", err)
	}

	// Return a termination error to close the WebSocket connection
	return &websocketTerminateError{message: "User quit"}
}
