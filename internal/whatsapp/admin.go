package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	waProto "go.mau.fi/whatsmeow/proto/waE2E"

	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/naseer2426/split-bot-whatsapp/internal/db"
)

const (
	WhitelistGroupCommand = "/whitelist-group"
	OnboardCommand        = "/onboard"
	ModeCommand           = "/mode"
	ChatMetaCommand       = "/chat-meta"
	HelpCommand           = "/help"
)

// AdminCmd runs an admin command; chatID is evt.Info.Chat.String() for the message (where the command was sent).
type AdminCmd func(chatID string, parts []string) string

var AdminCommands = map[string]AdminCmd{
	WhitelistGroupCommand: runWhitelistGroupCmd,
	OnboardCommand:        runOnboardCmd,
	ModeCommand:           runModeCmd,
	ChatMetaCommand:       runChatMetaCmd,
}

func init() {
	AdminCommands[HelpCommand] = runHelpCmd
}

const adminJSONPath = "internal/whatsapp/admin.json"

// AdminConfig represents the admin configuration structure
type AdminConfig struct {
	Admins []string `json:"admins"`
}

// LoadAdmins loads admin user IDs from internal/whatsapp/admin.json (relative to the process working directory).
func LoadAdmins() (map[string]bool, error) {
	adminData, err := os.ReadFile(adminJSONPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", adminJSONPath, err)
	}

	var config AdminConfig
	if err := json.Unmarshal(adminData, &config); err != nil {
		return nil, fmt.Errorf("parse %s: %w", adminJSONPath, err)
	}

	admins := make(map[string]bool, len(config.Admins))
	for _, admin := range config.Admins {
		admins[admin] = true
	}

	return admins, nil
}

func parseAdminCmd(text string) (cmd string, parts []string, ok bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", nil, false
	}
	parts = strings.Fields(text)
	if len(parts) == 0 {
		return "", nil, false
	}
	cmd = strings.ToLower(parts[0])
	_, ok = AdminCommands[cmd]
	return cmd, parts, ok
}

// isAdminMessage reports whether the sender is an admin and the trimmed message starts with a known admin command.
func (h *Handler) isAdminMessage(evt *events.Message) bool {
	senderID := cleanSenderID(evt.Info.Sender.String())
	senderIsAdmin := h.admins[senderID]
	_, _, adminCmd := parseAdminCmd(getMessageText(evt))
	return senderIsAdmin && adminCmd
}

// handleAdminMessage handles admin commands (DM or group).
func (h *Handler) handleAdminMessage(evt *events.Message) {

	cleanup := h.sendProcessing(context.Background(), evt)
	defer cleanup()

	messageText := getMessageText(evt)
	cmd, parts, _ := parseAdminCmd(messageText)

	run := AdminCommands[cmd]
	chatID := evt.Info.Chat.String()
	response := run(chatID, parts)

	_, err := h.client.SendMessage(context.Background(), evt.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(response),
		},
	})
	if err != nil {
		fmt.Printf("Error sending message: %v\n", err)
	}
}

func runHelpCmd(_ string, parts []string) string {
	if len(parts) != 1 {
		return "Usage: /help"
	}
	names := make([]string, 0, len(AdminCommands))
	for name := range AdminCommands {
		names = append(names, "- "+name)
	}
	sort.Strings(names)
	return "Available commands:\n\n" + strings.Join(names, "\n")
}

func runWhitelistGroupCmd(_ string, parts []string) string {
	if len(parts) != 3 {
		return "Error: /whitelist-group expects exactly 2 arguments after the command.\nExample: /whitelist-group 120363123456789012@g.us silent"
	}
	chatID, mode := parts[1], parts[2]
	if chatID == "" || mode == "" {
		return "chatID and mode must be non-empty."
	}

	if err := db.UpsertWhitelistedChat(db.GetDB(), chatID, mode); err != nil {
		return fmt.Sprintf("failed to upsert whitelisted chat: %v", err)
	}

	return fmt.Sprintf("Successfully saved chat %s with mode %s", chatID, mode)
}

func runOnboardCmd(chatID string, parts []string) string {
	if len(parts) != 2 {
		return "Error: /onboard expects exactly one argument (mode).\nExample: /onboard silent"
	}
	mode := parts[1]
	if mode == "" {
		return "Error: mode must be non-empty."
	}

	if err := db.UpsertWhitelistedChat(db.GetDB(), chatID, mode); err != nil {
		return fmt.Sprintf("Error onboarding chat: %v", err)
	}
	return fmt.Sprintf("Onboarded this chat (%s) with mode %q.", chatID, mode)
}

func runModeCmd(chatID string, parts []string) string {
	if len(parts) != 2 {
		return "Error: /mode expects exactly one argument.\nExample: /mode silent"
	}
	mode := parts[1]
	if mode == "" {
		return "Error: mode must be non-empty."
	}

	_, whitelisted, err := db.GetChatMeta(db.GetDB(), chatID)
	if err != nil {
		return fmt.Sprintf("Error checking chat: %v", err)
	}
	if !whitelisted {
		return fmt.Sprintf("This chat (%s) is not in the database. Ask Naseer to whitelist it before changing mode.", chatID)
	}

	if err := db.SetChatMode(db.GetDB(), chatID, mode); err != nil {
		return fmt.Sprintf("Error updating mode: %v", err)
	}
	return fmt.Sprintf("Updated this chat to mode %q.", mode)
}

func runChatMetaCmd(chatID string, parts []string) string {
	if len(parts) != 1 {
		return "Usage: /chat-meta"
	}
	row, whitelisted, err := db.GetWhatsappBotChatMeta(db.GetDB(), chatID)
	if err != nil {
		return fmt.Sprintf("Error loading chat meta: %v", err)
	}
	if !whitelisted {
		return fmt.Sprintf("chat (%s) not whitelisted yet", chatID)
	}
	out, err := json.MarshalIndent(row, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error encoding chat meta: %v", err)
	}
	return string(out)
}
