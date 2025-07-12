package web

import (
	"time"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type WSHistoryRequest struct {
	WSRequest
	ChannelID int  `json:"channel_id"`
	Limit     int  `json:"limit,omitempty"`
	Before    *int `json:"before,omitempty"`
	After     *int `json:"after,omitempty"`
}

type WSHistoryResponse struct {
	Messages []interface{} `json:"messages"`
	HasMore  bool          `json:"has_more"`
}

func (h *WebSocketHandler) HandleHistory(sess *chat.Session, data []byte) error {
	var req WSHistoryRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err
	}

	// Check if user is logged in
	if sess.UserID == nil {
		return sess.RespondError(req.ReqID, "Must be logged in to get message history", nil)
	}

	// Check if user is in the channel
	if !sess.IsInChannel(req.ChannelID) {
		return sess.RespondError(req.ReqID, "You must be in the channel to view its history", nil)
	}

	// Validate channel exists
	channel, err := models.GetChannelByID(h.db, req.ChannelID)
	if err != nil {
		return sess.RespondError(req.ReqID, "Database error", err)
	}
	if channel == nil {
		return sess.RespondError(req.ReqID, "Channel not found", nil)
	}

	// Set default limit if not provided
	if req.Limit <= 0 {
		req.Limit = 100
	}

	// Get message history with pagination
	historyOptions := models.MessageHistoryOptions{
		Limit:  req.Limit,
		Before: req.Before,
		After:  req.After,
	}

	messages, err := models.GetMessageHistory(h.db, req.ChannelID, historyOptions)
	if err != nil {
		return sess.RespondError(req.ReqID, "Failed to retrieve message history", err)
	}

	// Convert messages to WebSocket format
	var responseMessages []interface{}
	for _, msg := range messages {
		if msg.Event != "" && msg.Event != "message" {
			// Send as event
			eventMsg := WSEvent{
				Type:      "event",
				ChannelID: req.ChannelID,
				Event:     msg.Event,
				UserID:    msg.UserID,
				Nickname:  msg.Nickname,
				SentAt:    msg.SentAt.Format(time.RFC3339),
			}
			// For topic_change events, include the topic from the message content
			if msg.Event == "topic_change" && msg.Message != "" {
				eventMsg.Topic = &msg.Message
			}
			responseMessages = append(responseMessages, eventMsg)
		} else {
			// Send as regular message
			chatMsg := WSMessage{
				Type:      "message",
				ChannelID: req.ChannelID,
				Message:   msg.Message,
				IsPassive: msg.IsPassive,
				SentAt:    msg.SentAt.Format(time.RFC3339),
				UserID:    msg.UserID,
				Nickname:  msg.Nickname,
			}
			responseMessages = append(responseMessages, chatMsg)
		}
	}

	// Check if there are more messages available
	// We do this by trying to fetch one more message with the same constraints
	hasMore := false
	if len(messages) == req.Limit {
		checkOptions := historyOptions
		checkOptions.Limit = 1

		if req.Before != nil {
			// For "before" queries, check if there are older messages
			if len(messages) > 0 {
				oldestID := messages[len(messages)-1].ID
				checkOptions.Before = &oldestID
				checkOptions.After = nil
			}
		} else if req.After != nil {
			// For "after" queries, check if there are newer messages
			if len(messages) > 0 {
				newestID := messages[0].ID
				checkOptions.After = &newestID
				checkOptions.Before = nil
			}
		} else {
			// For recent message queries, check if there are older messages
			if len(messages) > 0 {
				oldestID := messages[len(messages)-1].ID
				checkOptions.Before = &oldestID
				checkOptions.After = nil
			}
		}

		checkMessages, err := models.GetMessageHistory(h.db, req.ChannelID, checkOptions)
		if err == nil && len(checkMessages) > 0 {
			hasMore = true
		}
	}

	response := WSHistoryResponse{
		Messages: responseMessages,
		HasMore:  hasMore,
	}

	return sess.RespondSuccess(req.ReqID, response)
}
