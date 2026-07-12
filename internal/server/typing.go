package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// typingRequest is the JSON body for POST /typing.
type typingRequest struct {
	ChatID string `json:"chat_id" binding:"required"`
}

func (s *Server) typingHandler(c *gin.Context) {
	var req typingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if err := s.handler.SendTyping(c.Request.Context(), req.ChatID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to set typing",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}
