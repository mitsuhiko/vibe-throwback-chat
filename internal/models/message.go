package models

import (
	"throwback-chat/internal/db"
	"time"
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

// MessageHistoryOptions represents pagination options for message history
type MessageHistoryOptions struct {
	Limit  int
	Before *int // Get messages before this message ID
	After  *int // Get messages after this message ID
}

// GetMessageHistory retrieves messages with pagination support
func GetMessageHistory(database *db.DB, channelID int, options MessageHistoryOptions) ([]*Message, error) {
	var messages []*Message
	var query string
	var args []interface{}

	// Default limit
	if options.Limit <= 0 || options.Limit > 500 {
		options.Limit = 100
	}

	// Base query
	baseQuery := `SELECT id, channel_id, user_id, sent_at, message, is_passive, event, nickname
				  FROM messages 
				  WHERE channel_id = ?`
	args = append(args, channelID)

	// Add pagination conditions
	if options.Before != nil && options.After != nil {
		// Get messages between two IDs
		query = baseQuery + ` AND id > ? AND id < ? ORDER BY sent_at DESC LIMIT ?`
		args = append(args, *options.After, *options.Before, options.Limit)
	} else if options.Before != nil {
		// Get messages before a specific ID
		query = baseQuery + ` AND id < ? ORDER BY sent_at DESC LIMIT ?`
		args = append(args, *options.Before, options.Limit)
	} else if options.After != nil {
		// Get messages after a specific ID
		query = baseQuery + ` AND id > ? ORDER BY sent_at ASC LIMIT ?`
		args = append(args, *options.After, options.Limit)
	} else {
		// Get most recent messages (default behavior)
		query = baseQuery + ` ORDER BY sent_at DESC LIMIT ?`
		args = append(args, options.Limit)
	}

	err := database.ReadDBX().Select(&messages, query, args...)

	// If we got messages after a specific ID, reverse them to maintain chronological order
	if options.After != nil && options.Before == nil {
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
	}

	return messages, err
}
