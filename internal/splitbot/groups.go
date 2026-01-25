package splitbot

import (
	"fmt"
)

// WhitelistGroupRequest represents the payload for whitelisting a group
type WhitelistGroupRequest struct {
	GroupID      string `json:"group_id"`
	PlatformType string `json:"platform_type"`
}

// WhitelistGroup whitelists a group via the API
// It uses the SPLIT_BOT_URL environment variable to determine the API base URL
func WhitelistGroup(groupID string) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	req := WhitelistGroupRequest{
		GroupID:      groupID,
		PlatformType: "WHATSAPP",
	}

	resp, err := client.R().
		SetBody(req).
		Post("/whitelisted-chats")

	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("API returned non-success status: %d, body: %s", resp.StatusCode(), resp.String())
	}

	return nil
}
