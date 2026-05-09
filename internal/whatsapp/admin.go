package whatsapp

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	waProto "go.mau.fi/whatsmeow/proto/waE2E"

	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/naseer2426/split-bot-whatsapp/internal/splitbot"
)

const (
	WhitelistCommand = "/whitelist"
)

//go:embed admin.json
var adminJSONData []byte

// AdminConfig represents the admin configuration structure
type AdminConfig struct {
	Admins []string `json:"admins"`
}

// LoadAdmins loads admin user IDs from admin.json file
func LoadAdmins() (map[string]bool, error) {
	// First try to use embedded data
	var adminData []byte
	var err error

	if len(adminJSONData) > 0 {
		adminData = adminJSONData
	} else {
		// Fallback: try to read from file system
		// Try multiple possible paths
		possiblePaths := []string{
			filepath.Join("internal", "whatsapp", "admin.json"),
			filepath.Join(".", "internal", "whatsapp", "admin.json"),
			"admin.json",
		}

		for _, path := range possiblePaths {
			adminData, err = os.ReadFile(path)
			if err == nil {
				break
			}
		}

		if err != nil {
			return nil, fmt.Errorf("failed to read admin.json: %w", err)
		}
	}

	var config AdminConfig
	if err := json.Unmarshal(adminData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse admin.json: %w", err)
	}

	admins := make(map[string]bool)
	for _, admin := range config.Admins {
		admins[admin] = true
	}

	return admins, nil
}

// isAdminMessage checks if a message is from an admin in a private chat
func (h *WaHandler) isAdminMessage(evt *events.Message) bool {
	// Check if message is from a private chat (not a group)
	// Groups have "g.us" as the server part
	if evt.Info.Chat.Server == "g.us" {
		return false
	}

	// Check if the sender is in the admins list
	senderID := cleanSenderID(evt.Info.Sender.String())
	return h.admins[senderID]
}

// handleAdminMessage handles admin messages (empty for now)
func (h *WaHandler) handleAdminMessage(evt *events.Message) {
	messageText := getMessageText(evt)
	response := "Unrecognized command"

	if strings.Contains(strings.ToLower(messageText), strings.ToLower(WhitelistCommand)) {
		response = h.whitelistCommand(messageText)
	}

	_, err := h.client.SendMessage(context.Background(), evt.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(response),
		},
	})
	if err != nil {
		fmt.Printf("Error sending message: %v\n", err)
	}
}

func (h *WaHandler) whitelistCommand(messageText string) string {
	// Trim whitespace from the message
	messageText = strings.TrimSpace(messageText)
	
	// Split by whitespace
	parts := strings.Fields(messageText)
	
	// Check if we have the correct number of parts
	if len(parts) < 2 {
		return "Error: Invalid format. Expected format: /whitelist <group_id>\nExample: /whitelist 120363123456789012@g.us"
	}
	
	// Extract group_id (could be multiple words, but typically it's just one)
	groupID := parts[1]
	
	// Validate group_id is not empty
	if groupID == "" {
		return "Error: Group ID cannot be empty. Expected format: /whitelist <group_id>"
	}
	
	// Call the WhitelistGroup API
	if err := splitbot.WhitelistGroup(groupID); err != nil {
		return fmt.Sprintf("Error: %s", err.Error())
	}
	
	// Return success message
	return fmt.Sprintf("Successfully whitelisted %s", groupID)
}
