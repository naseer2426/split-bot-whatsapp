package splitbot

import (
	"fmt"
)

type ImageBase64 struct {
	Data string `json:"data"`
	MType string `json:"mtype"`
}

// ProcessMessageRequest represents the payload for processing a message
type ProcessMessageRequest struct {
	Message      string `json:"message"`
	GroupID      string `json:"group_id"`
	Sender       string `json:"sender"`
	PlatformType string `json:"platform_type"`
	ImageBase64 *ImageBase64 `json:"image_base64"`
}

// ProcessMessageResponse represents the response from processing a message
type ProcessMessageResponse struct {
	Response string  `json:"response"`
	Error    *string `json:"error"`
}

// ProcessMessage processes a message via the API
// It uses the SPLIT_BOT_URL environment variable to determine the API base URL
func ProcessMessage(req ProcessMessageRequest) (*ProcessMessageResponse, error) {
	req.PlatformType = "WHATSAPP"
	client, err := getClient()
	if err != nil {
		return nil, err
	}

	var response ProcessMessageResponse
	resp, err := client.R().
		SetBody(req).
		SetResult(&response).
		Post("/process_message")

	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("API returned non-success status: %d, body: %s", resp.StatusCode(), resp.String())
	}

	if response.Error != nil && *response.Error != "" {
		return nil, fmt.Errorf("API returned error: %s", *response.Error)
	}

	return &response, nil
}
