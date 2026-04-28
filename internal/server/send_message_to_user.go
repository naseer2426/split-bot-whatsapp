package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// sendMessageToUserRequest represents the request body for sending a message to a user (1:1)
type sendMessageToUserRequest struct {
	Message string `json:"message" binding:"required"`
	UserID  string `json:"user_id" binding:"required"`
}

// sendMessageToUserHandler handles POST requests to send messages to a WhatsApp user
func (s *Server) sendMessageToUserHandler(c *gin.Context) {
	var req sendMessageToUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if err := s.handler.SendMessageToUser(req.Message, req.UserID); err != nil {
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
