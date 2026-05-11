package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// pollCreateRequest is the JSON body for POST /poll/create.
type pollCreateRequest struct {
	Title    string   `json:"title" binding:"required"`
	Options  []string `json:"options" binding:"required,min=1"`
	GroupID  string   `json:"group_id" binding:"required"`
}

// pollCreateHandler creates a WhatsApp poll in the given group via whatsapp.Handler.SendPoll.
func (s *Server) pollCreateHandler(c *gin.Context) {
	var req pollCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	pollRow, err := s.handler.SendPoll(c.Request.Context(), req.Title, req.Options, req.GroupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create poll",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"poll_id":      pollRow.ID,
		"message_keys": pollRow.MessageKeys,
	})
}

// pollStatusHandler returns aggregated voters per option for a collective poll via whatsapp.Handler.GetPollStatus.
func (s *Server) pollStatusHandler(c *gin.Context) {
	pollIDStr := c.Query("poll_id")
	pollID, err := strconv.Atoi(pollIDStr)
	if err != nil || pollID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid or missing poll_id query parameter",
			"details": "poll_id must be a positive integer",
		})
		return
	}

	options, err := s.handler.GetPollStatus(c.Request.Context(), pollID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Poll not found",
				"details": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to load poll status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"options": options,
	})
}
