package models

import (
	"database/sql"
	"errors"
	"strings"
	"throwback-chat/internal/db"
)

type Channel struct {
	ID    int    `json:"id" db:"id"`
	Name  string `json:"name" db:"name"`
	Topic string `json:"topic" db:"topic"`
}

// NormalizeChannelName ensures channel names start with '#' and are lowercase
func NormalizeChannelName(name string) string {
	// Remove leading/trailing spaces
	name = strings.TrimSpace(name)

	// Convert to lowercase
	name = strings.ToLower(name)

	// Add '#' prefix if not present
	if !strings.HasPrefix(name, "#") {
		name = "#" + name
	}

	return name
}

// ValidateChannelName checks if a channel name is valid
func ValidateChannelName(name string) error {
	if name == "" {
		return errors.New("channel name cannot be empty")
	}

	// Normalize the name
	normalized := NormalizeChannelName(name)

	// Check length (including #)
	if len(normalized) < 2 {
		return errors.New("channel name must be at least 1 character long")
	}

	if len(normalized) > 50 {
		return errors.New("channel name cannot exceed 49 characters")
	}

	// Check for valid characters (alphanumeric, dash, underscore)
	for i, r := range normalized {
		if i == 0 && r == '#' {
			continue // Skip the '#' prefix
		}
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return errors.New("channel name can only contain alphanumeric characters, dashes, and underscores")
		}
	}

	return nil
}

func GetChannelByName(database *db.DB, name string) (*Channel, error) {
	// Normalize the channel name for lookup
	normalizedName := NormalizeChannelName(name)

	var channel Channel
	err := database.ReadDBX().Get(&channel, "SELECT id, name, topic FROM channels WHERE name = ?", normalizedName)
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
	// Validate the channel name
	if err := ValidateChannelName(name); err != nil {
		return nil, err
	}

	// Normalize the channel name
	normalizedName := NormalizeChannelName(name)

	result, err := database.WriteDB().Exec("INSERT INTO channels (name, topic) VALUES (?, '')", normalizedName)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Channel{
		ID:    int(id),
		Name:  normalizedName,
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

func UpdateChannelTopic(database *db.DB, channelID int, topic string) error {
	_, err := database.WriteDB().Exec("UPDATE channels SET topic = ? WHERE id = ?", topic, channelID)
	return err
}

// GetChannelUserCount returns the current number of users in a channel
func GetChannelUserCount(database *db.DB, channelID int) (int, error) {
	// Count the balance of join/leave events to determine current user count
	var joinCount, leaveCount int

	err := database.ReadDBX().Get(&joinCount, "SELECT COUNT(*) FROM messages WHERE channel_id = ? AND event = 'joined'", channelID)
	if err != nil {
		return 0, err
	}

	err = database.ReadDBX().Get(&leaveCount, "SELECT COUNT(*) FROM messages WHERE channel_id = ? AND event = 'left'", channelID)
	if err != nil {
		return 0, err
	}

	userCount := joinCount - leaveCount
	if userCount < 0 {
		userCount = 0
	}

	return userCount, nil
}

// GetAllChannelsWithInfo returns all channels with their user counts
func GetAllChannelsWithInfo(database *db.DB) ([]ChannelInfo, error) {
	var channels []Channel
	err := database.ReadDBX().Select(&channels, "SELECT id, name, topic FROM channels ORDER BY name")
	if err != nil {
		return nil, err
	}

	var channelInfos []ChannelInfo
	for _, channel := range channels {
		userCount, err := GetChannelUserCount(database, channel.ID)
		if err != nil {
			return nil, err
		}

		channelInfos = append(channelInfos, ChannelInfo{
			ID:        channel.ID,
			Name:      channel.Name,
			Topic:     channel.Topic,
			UserCount: userCount,
		})
	}

	return channelInfos, nil
}

// GetUserChannels returns the channels a user is currently in
func GetUserChannels(database *db.DB, userID int) ([]ChannelInfo, error) {
	// Get all channels where the user has more joins than leaves
	query := `
		SELECT DISTINCT c.id, c.name, c.topic
		FROM channels c
		JOIN messages m ON c.id = m.channel_id
		WHERE m.user_id = ? AND m.event IN ('joined', 'left')
		GROUP BY c.id, c.name, c.topic
		HAVING SUM(CASE WHEN m.event = 'joined' THEN 1 ELSE -1 END) > 0
		ORDER BY c.name
	`

	var channels []Channel
	err := database.ReadDBX().Select(&channels, query, userID)
	if err != nil {
		return nil, err
	}

	var channelInfos []ChannelInfo
	for _, channel := range channels {
		userCount, err := GetChannelUserCount(database, channel.ID)
		if err != nil {
			return nil, err
		}

		channelInfos = append(channelInfos, ChannelInfo{
			ID:        channel.ID,
			Name:      channel.Name,
			Topic:     channel.Topic,
			UserCount: userCount,
		})
	}

	return channelInfos, nil
}

// DeleteEmptyChannel removes a channel if it has no users
func DeleteEmptyChannel(database *db.DB, channelID int) error {
	userCount, err := GetChannelUserCount(database, channelID)
	if err != nil {
		return err
	}

	if userCount == 0 {
		// Delete channel and all associated data
		tx, err := database.WriteDB().Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// Delete ops first (foreign key constraint)
		_, err = tx.Exec("DELETE FROM ops WHERE channel_id = ?", channelID)
		if err != nil {
			return err
		}

		// Delete messages
		_, err = tx.Exec("DELETE FROM messages WHERE channel_id = ?", channelID)
		if err != nil {
			return err
		}

		// Delete channel
		_, err = tx.Exec("DELETE FROM channels WHERE id = ?", channelID)
		if err != nil {
			return err
		}

		return tx.Commit()
	}

	return nil
}

// ChannelInfo represents channel information with user count
type ChannelInfo struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Topic     string `json:"topic"`
	UserCount int    `json:"user_count"`
}

// ChannelUser represents a user in a channel with their status
type ChannelUser struct {
	ID       int    `json:"id"`
	Nickname string `json:"nickname"`
	IsServ   bool   `json:"is_serv"`
	IsOp     bool   `json:"is_op"`
}

// GetChannelUsers returns all users currently in a channel
func GetChannelUsers(database *db.DB, channelID int) ([]ChannelUser, error) {
	// Get users who have more joins than leaves in this channel
	query := `
		SELECT DISTINCT u.id, u.nickname, u.is_serv,
		       COALESCE(ops.user_id IS NOT NULL, 0) as is_op
		FROM users u
		JOIN messages m ON u.id = m.user_id
		LEFT JOIN ops ON u.id = ops.user_id AND ops.channel_id = ?
		WHERE m.channel_id = ? AND m.event IN ('joined', 'left')
		GROUP BY u.id, u.nickname, u.is_serv
		HAVING SUM(CASE WHEN m.event = 'joined' THEN 1 ELSE -1 END) > 0
		ORDER BY 
		    COALESCE(ops.user_id IS NOT NULL, 0) DESC,  -- Ops first
		    u.is_serv DESC,                             -- Service users next
		    u.nickname ASC                              -- Then alphabetical
	`

	var users []ChannelUser
	err := database.ReadDBX().Select(&users, query, channelID, channelID)
	if err != nil {
		return nil, err
	}

	return users, nil
}
