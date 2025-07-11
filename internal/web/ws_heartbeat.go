package web

import (
	"throwback-chat/internal/chat"
)

type HeartbeatResponse struct {
	Timestamp int64 `json:"timestamp"`
}

func (h *WebSocketHandler) HandleHeartbeat(sess *chat.Session, reqID string) error {
	return sess.SendMessage(NewWSResponse(reqID, true, "", HeartbeatResponse{
		Timestamp: sess.LastHeartbeat.Unix(),
	}))
}
