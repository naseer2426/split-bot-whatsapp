package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// sendMessageToGroupRequest represents the request body for sending a message to a group
type sendMessageToGroupRequest struct {
	Message string `json:"message" binding:"required"`
	GroupID string `json:"group_id" binding:"required"`
}

// sendMessageToGroupHandler handles POST requests to send messages to WhatsApp groups
func (s *Server) sendMessageToGroupHandler(c *gin.Context) {
	var req sendMessageToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Send message using the handler
	if err := s.handler.SendMessageToGroup(req.Message, req.GroupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send message",
			"details": err.Error(),
		})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Message sent successfully",
	})
}
