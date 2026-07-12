package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/naseer2426/split-bot-whatsapp/internal/whatsapp"
)

// sendMediaToChatRequest is the JSON body for POST /send_media_to_chat.
type sendMediaToChatRequest struct {
	ChatID     string `json:"chat_id" binding:"required"`
	MediaType  string `json:"media_type" binding:"required"` // image|video|audio|document
	DataBase64 string `json:"data_base64"`
	FileURL    string `json:"file_url"`
	Mime       string `json:"mime"`
	Caption    string `json:"caption"`
	Filename   string `json:"filename"`
	ReplyTo    string `json:"reply_to"`
}

func (s *Server) sendMediaToChatHandler(c *gin.Context) {
	var req sendMediaToChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if req.DataBase64 == "" && req.FileURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": "either data_base64 or file_url is required",
		})
		return
	}

	err := s.handler.SendMediaToChat(c.Request.Context(), whatsapp.SendMediaToChatParams{
		ChatID:     req.ChatID,
		MediaType:  req.MediaType,
		DataBase64: req.DataBase64,
		FileURL:    req.FileURL,
		Mime:       req.Mime,
		Caption:    req.Caption,
		Filename:   req.Filename,
		ReplyTo:    req.ReplyTo,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if isClientMediaError(err) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{
			"error":   "Failed to send media",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Media sent successfully",
	})
}

func isClientMediaError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, sub := range []string{
		"invalid chat JID",
		"invalid data_base64",
		"either data_base64 or file_url",
		"unsupported media_type",
		"exceeds max size",
		"decoded to empty",
		"returned empty body",
		"invalid file_url",
	} {
		if strings.Contains(msg, sub) {
			return true
		}
	}
	return false
}
