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

	// Get users from active sessions (not database reconstruction)
	var users []models.ChannelUser
	activeSessions := h.sessions.GetSessions()

	for _, activeSession := range activeSessions {
		// Only include logged-in users who are in this channel
		if activeSession.UserID != nil && activeSession.IsInChannel(req.ChannelID) {
			// Get user info from database
			var user models.User
			err := h.db.ReadDBX().Get(&user, "SELECT id, nickname, is_serv FROM users WHERE id = ?", *activeSession.UserID)
			if err != nil {
				continue // Skip this user if we can't get their info
			}

			// Check if user is an operator
			var isOp bool
			err = h.db.ReadDBX().Get(&isOp, "SELECT COUNT(*) > 0 FROM ops WHERE user_id = ? AND channel_id = ?", *activeSession.UserID, req.ChannelID)
			if err != nil {
				isOp = false // Default to not op if query fails
			}

			users = append(users, models.ChannelUser{
				ID:       user.ID,
				Nickname: user.Nickname,
				IsServ:   user.IsServ,
				IsOp:     isOp,
			})
		}
	}

	// Send response
	response := ChannelUsersResponse{
		Users: users,
	}

	return sess.RespondSuccess(req.ReqID, response)
}
