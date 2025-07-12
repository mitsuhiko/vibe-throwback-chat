package web

import (
	"log"
	"time"

	"throwback-chat/internal/chat"
	"throwback-chat/internal/models"
)

type WSTopicRequest struct {
	WSRequest
	ChannelID int    `json:"channel_id"`
	Topic     string `json:"topic"`
}

type WSTopicResponse struct {
	ChannelID int    `json:"channel_id"`
	Topic     string `json:"topic"`
}

func (h *WebSocketHandler) HandleTopic(sess *chat.Session, data []byte) error {
	var req WSTopicRequest
	if err := DecodeWSData(sess, data, "", &req); err != nil {
		return err
	}

	// Check if user is logged in
	if sess.UserID == nil {
		return sess.RespondError(req.ReqID, "Must be logged in to change channel topic", nil)
	}

	// Validate required fields
	if req.ChannelID == 0 {
		return sess.RespondError(req.ReqID, "Channel ID is required", nil)
	}

	// Check if the requesting user is an operator of the channel
	isOp, err := models.IsUserOp(h.db, *sess.UserID, req.ChannelID)
	if err != nil {
		return sess.RespondError(req.ReqID, "Database error", err)
	}
	if !isOp {
		return sess.RespondError(req.ReqID, "You must be an operator to change the channel topic", nil)
	}

	// Verify the channel exists
	channel, err := models.GetChannelByID(h.db, req.ChannelID)
	if err != nil {
		return sess.RespondError(req.ReqID, "Database error", err)
	}
	if channel == nil {
		return sess.RespondError(req.ReqID, "Channel not found", nil)
	}

	// Update the channel topic in the database
	err = models.UpdateChannelTopic(h.db, req.ChannelID, req.Topic)
	if err != nil {
		return sess.RespondError(req.ReqID, "Failed to update topic", err)
	}

	// Create topic change event message
	topicMessage := req.Topic
	if topicMessage == "" {
		topicMessage = "Topic cleared"
	}

	// Create topic change event in database
	_, err = models.CreateMessage(h.db, &req.ChannelID, *sess.UserID, topicMessage, "topic_change", *sess.Nickname, false)
	if err != nil {
		log.Printf("Failed to create topic change message: %v", err)
	}

	// Broadcast topic change event to all users in the channel
	topicEvent := WSEvent{
		Type:      "event",
		ChannelID: req.ChannelID,
		Event:     "topic_change",
		UserID:    *sess.UserID,
		Nickname:  *sess.Nickname,
		SentAt:    time.Now().Format(time.RFC3339),
		Topic:     &req.Topic,
	}
	h.sessions.BroadcastToChannel(req.ChannelID, topicEvent)

	log.Printf("User %s changed topic for channel %s (ID: %d) to: %s",
		*sess.Nickname, channel.Name, channel.ID, req.Topic)

	return sess.RespondSuccess(req.ReqID, WSTopicResponse{
		ChannelID: req.ChannelID,
		Topic:     req.Topic,
	})
}
