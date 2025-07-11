package models

import (
	"database/sql"
	"throwback-chat/internal/db"
)

type Channel struct {
	ID    int    `json:"id" db:"id"`
	Name  string `json:"name" db:"name"`
	Topic string `json:"topic" db:"topic"`
}

func GetChannelByName(database *db.DB, name string) (*Channel, error) {
	var channel Channel
	err := database.ReadDBX().Get(&channel, "SELECT id, name, topic FROM channels WHERE name = ?", name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &channel, nil
}

func GetChannelByID(database *db.DB, id int) (*Channel, error) {
	var channel Channel
	err := database.ReadDBX().Get(&channel, "SELECT id, name, topic FROM channels WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &channel, nil
}

func CreateChannel(database *db.DB, name string) (*Channel, error) {
	result, err := database.WriteDB().Exec("INSERT INTO channels (name, topic) VALUES (?, '')", name)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Channel{
		ID:    int(id),
		Name:  name,
		Topic: "",
	}, nil
}

func IsChannelEmpty(database *db.DB, channelID int) (bool, error) {
	var count int
	err := database.ReadDBX().Get(&count, "SELECT COUNT(*) FROM messages WHERE channel_id = ? AND event IN ('joined', 'left')", channelID)
	if err != nil {
		return false, err
	}

	// Get the latest join/leave events to determine if channel is empty
	var joined, left int
	database.ReadDBX().Get(&joined, "SELECT COUNT(*) FROM messages WHERE channel_id = ? AND event = 'joined'", channelID)
	database.ReadDBX().Get(&left, "SELECT COUNT(*) FROM messages WHERE channel_id = ? AND event = 'left'", channelID)

	return joined <= left, nil
}

func MakeUserOp(database *db.DB, userID, channelID, grantedByUserID int) error {
	_, err := database.WriteDB().Exec(
		"INSERT OR REPLACE INTO ops (user_id, channel_id, granted_by_user_id) VALUES (?, ?, ?)",
		userID, channelID, grantedByUserID,
	)
	return err
}

func IsUserOp(database *db.DB, userID, channelID int) (bool, error) {
	var count int
	err := database.ReadDBX().Get(&count, "SELECT COUNT(*) FROM ops WHERE user_id = ? AND channel_id = ?", userID, channelID)
	return count > 0, err
}
