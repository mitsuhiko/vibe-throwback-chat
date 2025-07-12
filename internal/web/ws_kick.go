package web

import (
	"log"
	"time"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type WSKickRequest struct {
	WSRequest
	UserID    int    `json:"user_id"`
	ChannelID int    `json:"channel_id"`
	Reason    string `json:"reason,omitempty"`
}

type WSKickResponse struct {
	UserID    int    `json:"user_id"`
	ChannelID int    `json:"channel_id"`
	Reason    string `json:"reason,omitempty"`
}

func (h *WebSocketHandler) HandleKick(sess *chat.Session, data []byte) error {
	var req WSKickRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err
	}

	// Check if user is logged in
	if sess.UserID == nil {
		return sess.RespondError(req.ReqID, "Must be logged in to kick users")
	}

	// Validate required fields
	if req.UserID == 0 {
		return sess.RespondError(req.ReqID, "User ID is required")
	}
	if req.ChannelID == 0 {
		return sess.RespondError(req.ReqID, "Channel ID is required")
	}

	// Check if the requesting user is an operator of the channel
	isOp, err := models.IsUserOp(h.db, *sess.UserID, req.ChannelID)
	if err != nil {
		log.Printf("Failed to check operator status: %v", err)
		return sess.RespondError(req.ReqID, "Database error")
	}
	if !isOp {
		return sess.RespondError(req.ReqID, "You must be an operator to kick users")
	}

	// Verify the channel exists
	channel, err := models.GetChannelByID(h.db, req.ChannelID)
	if err != nil {
		log.Printf("Failed to get channel by ID: %v", err)
		return sess.RespondError(req.ReqID, "Database error")
	}
	if channel == nil {
		return sess.RespondError(req.ReqID, "Channel not found")
	}

	// Get the target user information
	targetUser := &models.User{}
	err = h.db.ReadDBX().Get(targetUser, "SELECT id, nickname, is_serv FROM users WHERE id = ?", req.UserID)
	if err != nil {
		log.Printf("Failed to get target user: %v", err)
		return sess.RespondError(req.ReqID, "Target user not found")
	}

	// Prevent kicking ChanServ or other service users
	if targetUser.IsServ {
		return sess.RespondError(req.ReqID, "Cannot kick service users")
	}

	// Prevent self-kick (though this might be allowed in some IRC implementations)
	if req.UserID == *sess.UserID {
		return sess.RespondError(req.ReqID, "Cannot kick yourself")
	}

	// Find all sessions for the target user
	targetSessions := h.sessions.GetSessionsByUserID(req.UserID)

	// Remove the target user from the channel in all their sessions
	kicked := false
	for _, targetSession := range targetSessions {
		if targetSession.IsInChannel(req.ChannelID) {
			targetSession.LeaveChannel(req.ChannelID)
			kicked = true
		}
	}

	if !kicked {
		return sess.RespondError(req.ReqID, "User is not in the channel")
	}

	// Create kick event message
	kickMessage := req.Reason
	if kickMessage == "" {
		kickMessage = "Kicked"
	}

	// Create kick event in database
	_, err = models.CreateMessage(h.db, &req.ChannelID, req.UserID, kickMessage, "kicked", targetUser.Nickname, false)
	if err != nil {
		log.Printf("Failed to create kick message: %v", err)
	}

	// Broadcast kick event to all users in the channel
	kickEvent := WSEvent{
		Type:      "event",
		ChannelID: req.ChannelID,
		Event:     "kicked",
		UserID:    req.UserID,
		Nickname:  targetUser.Nickname,
		SentAt:    time.Now().Format(time.RFC3339),
	}
	h.sessions.BroadcastToChannel(req.ChannelID, kickEvent)

	log.Printf("User %s kicked user %s from channel %s (ID: %d). Reason: %s",
		*sess.Nickname, targetUser.Nickname, channel.Name, channel.ID, kickMessage)

	return sess.RespondSuccess(req.ReqID, WSKickResponse{
		UserID:    req.UserID,
		ChannelID: req.ChannelID,
		Reason:    kickMessage,
	})
}
