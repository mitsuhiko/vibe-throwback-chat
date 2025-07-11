package web

import (
	"log"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type WSLogoutRequest struct {
	WSRequest
	DyingMessage string `json:"dying_message,omitempty"`
}

type WSLogoutResponse struct {
	Message      string `json:"message"`
	DyingMessage string `json:"dying_message,omitempty"`
}

func (h *WebSocketHandler) HandleLogout(sess *chat.Session, data []byte) error {
	var req WSLogoutRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err // This will terminate the websocket connection
	}

	if sess.UserID == nil {
		return sess.SendMessage(NewWSResponse(req.ReqID, false, "Not logged in", nil))
	}

	nickname := ""
	if sess.Nickname != nil {
		nickname = *sess.Nickname
	}

	// Emit logout events to all channels user was in
	userChannels := sess.GetChannels()
	for _, channelID := range userChannels {
		// Create leave event in database
		leaveMessage := req.DyingMessage
		if leaveMessage == "" {
			leaveMessage = "Logged out"
		}

		models.CreateMessage(h.db, &channelID, *sess.UserID, leaveMessage, "left", nickname, false)

		// Broadcast leave event to channel
		leaveEvent := WSEvent{
			Type:      "event",
			ChannelID: channelID,
			Event:     "left",
			UserID:    *sess.UserID,
			Nickname:  nickname,
			SentAt:    "",
		}
		h.sessions.BroadcastToChannel(channelID, leaveEvent)

		// Remove from channel subscription
		sess.LeaveChannel(channelID)
	}

	log.Printf("User %s logged out from session %s", nickname, sess.ID)

	// Clear user from session
	sess.ClearUser()

	logoutResponse := WSLogoutResponse{
		Message: "Logged out successfully",
	}

	if req.DyingMessage != "" {
		logoutResponse.DyingMessage = req.DyingMessage
	}

	return sess.SendMessage(NewWSResponse(req.ReqID, true, "", logoutResponse))
}
