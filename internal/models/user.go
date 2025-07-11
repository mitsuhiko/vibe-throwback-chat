package models

import (
	"database/sql"
	"fmt"

	"throwback-chat/internal/db"
)

type User struct {
	ID       int    `json:"id"`
	Nickname string `json:"nickname"`
	IsServ   bool   `json:"is_serv"`
}

func CreateOrUpdateUser(database *db.DB, nickname string) (*User, error) {
	// First try to get existing user
	user, err := GetUserByNickname(database, nickname)
	if err != nil || user != nil {
		return user, err
	}

	// User doesn't exist, use REPLACE to handle race conditions
	result, err := database.WriteDB().Exec("REPLACE INTO users (nickname, is_serv) VALUES (?, FALSE)", nickname)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	return &User{
		ID:       int(id),
		Nickname: nickname,
		IsServ:   false,
	}, nil
}

func GetUserByNickname(database *db.DB, nickname string) (*User, error) {
	var user User
	err := database.ReadDBX().Get(&user, "SELECT id, nickname, is_serv FROM users WHERE nickname = ?", nickname)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}
