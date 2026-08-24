package hermes

import (
	"fmt"
)

type MediaItem struct {
	Type     string `json:"type"` // "image" | "document" | "video" | "audio"
	Data     string `json:"data"` // base64
	Mime     string `json:"mime,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type MessageRequest struct {
	Sender    string      `json:"sender"`
	ChatID    string      `json:"chat_id"`
	Text      string      `json:"text"`
	Media     []MediaItem `json:"media,omitempty"`
	MessageID string      `json:"message_id,omitempty"`
	ReplyTo   string      `json:"reply_to,omitempty"`
	IsMention bool        `json:"is_mention,omitempty"`
}

type MessageResponse struct {
	OK bool `json:"ok"`
}

// SendMessage forwards an inbound WhatsApp message to the Hermes whatsapp_internal API.
func SendMessage(req MessageRequest) error {
	client := getClient()

	var response MessageResponse
	resp, err := client.R().
		SetBody(req).
		SetResult(&response).
		Post("/message")

	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("API returned non-success status: %d, body: %s", resp.StatusCode(), resp.String())
	}

	if !response.OK {
		return fmt.Errorf("API returned ok=false, body: %s", resp.String())
	}

	return nil
}
