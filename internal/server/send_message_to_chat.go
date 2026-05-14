package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// sendMessageToChatRequest is the JSON body for POST /send_message_to_chat.
// chat_id must be a full WhatsApp JID (e.g. user@…, group@g.us).
type sendMessageToChatRequest struct {
	Message string `json:"message" binding:"required"`
	ChatID  string `json:"chat_id" binding:"required"`
}

func (s *Server) sendMessageToChatHandler(c *gin.Context) {
	var req sendMessageToChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if err := s.handler.SendMessageToChat(req.Message, req.ChatID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send message",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Message sent successfully",
	})
}
