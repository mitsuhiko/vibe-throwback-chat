package models

import (
	"time"
	"throwback-chat/internal/db"
)

type Message struct {
	ID        int       `json:"id" db:"id"`
	ChannelID *int      `json:"channel_id" db:"channel_id"`
	UserID    int       `json:"user_id" db:"user_id"`
	SentAt    time.Time `json:"sent_at" db:"sent_at"`
	Message   string    `json:"message" db:"message"`
	IsPassive bool      `json:"is_passive" db:"is_passive"`
	Event     string    `json:"event" db:"event"`
	Nickname  string    `json:"nickname" db:"nickname"`
}

func CreateMessage(database *db.DB, channelID *int, userID int, message, event, nickname string, isPassive bool) (*Message, error) {
	query := `INSERT INTO messages (channel_id, user_id, message, event, nickname, is_passive) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	
	result, err := database.WriteDB().Exec(query, channelID, userID, message, event, nickname, isPassive)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:        int(id),
		ChannelID: channelID,
		UserID:    userID,
		SentAt:    time.Now(),
		Message:   message,
		IsPassive: isPassive,
		Event:     event,
		Nickname:  nickname,
	}, nil
}

func GetRecentMessages(database *db.DB, channelID int, limit int) ([]*Message, error) {
	var messages []*Message
	query := `SELECT id, channel_id, user_id, sent_at, message, is_passive, event, nickname
			  FROM messages 
			  WHERE channel_id = ? 
			  ORDER BY sent_at DESC 
			  LIMIT ?`
	
	err := database.ReadDBX().Select(&messages, query, channelID, limit)
	return messages, err
}