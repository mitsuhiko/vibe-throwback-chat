package web

type WSMessage struct {
	Type      string `json:"type"`
	ChannelID int    `json:"channel_id"`
	Message   string `json:"message"`
	IsPassive bool   `json:"is_passive"`
	SentAt    string `json:"sent_at"`
	UserID    int    `json:"user_id"`
	Nickname  string `json:"nickname"`
}