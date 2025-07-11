package web

import (
	"log"

	"throwback-chat/internal/chat"
)

type LogoutRequest struct {
	Cmd          string `json:"cmd"`
	ReqID        string `json:"req_id"`
	DyingMessage string `json:"dying_message,omitempty"`
}

type LogoutResponse struct {
	Message      string `json:"message"`
	DyingMessage string `json:"dying_message,omitempty"`
}

func (h *WebSocketHandler) HandleLogout(sess *chat.Session, data []byte) error {
	var req LogoutRequest
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

	log.Printf("User %s logged out from session %s", nickname, sess.ID)

	// Clear user from session
	sess.ClearUser()

	logoutResponse := LogoutResponse{
		Message: "Logged out successfully",
	}

	if req.DyingMessage != "" {
		logoutResponse.DyingMessage = req.DyingMessage
	}

	return sess.SendMessage(NewWSResponse(req.ReqID, true, "", logoutResponse))
}
