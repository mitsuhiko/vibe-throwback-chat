package web

import (
	"throwback-chat/internal/chat"
)

type WSHeartbeatResponse struct {
	Timestamp int64 `json:"timestamp"`
}

func (h *WebSocketHandler) HandleHeartbeat(sess *chat.Session, reqID string) error {
	return sess.SendMessage(NewWSResponse(reqID, true, "", WSHeartbeatResponse{
		Timestamp: sess.LastHeartbeat.Unix(),
	}))
}
