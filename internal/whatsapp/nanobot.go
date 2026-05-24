package whatsapp

import (
	"context"
	"fmt"

	"github.com/naseer2426/split-bot-whatsapp/internal/nanobot"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (h *Handler) handleNanobotMode(evt *events.Message) string {
	messageText := getMessageText(evt)
	if messageText == "" {
		return ""
	}

	if !h.shouldProcessNanobotMsg(evt, messageText) {
		fmt.Printf("Group message from %s doesn't include bot_name '%s', skipping...\n", evt.Info.Sender, h.botName)
		return ""
	}

	messageText = h.nanobotMessageText(messageText)
	if messageText == "" {
		return ""
	}

	// we don't send the stopTyping message here because the client will handle it when nanobot replies asynchronously
	_ = h.sendProcessing(context.Background(), evt)

	req := nanobot.MessageRequest{
		Sender: evt.Info.Sender.ToNonAD().String(),
		ChatID: evt.Info.Chat.ToNonAD().String(),
		Text:   messageText,
	}

	if err := nanobot.SendMessage(req); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	// Nanobot replies asynchronously via POST /send_message_to_chat.
	return ""
}

func (h *Handler) shouldProcessNanobotMsg(evt *events.Message, messageText string) bool {
	if evt.Info.Chat.Server != types.GroupServer {
		return true
	}
	return h.messageContainsBotName(messageText)
}

// nanobotMessageText returns the text to forward to nanobot, stripping @botName only for slash commands.
func (h *Handler) nanobotMessageText(message string) string {
	if !h.isBotNameSlashCommandMessage(message) {
		return message
	}
	return h.messageWithoutBotNameMention(message)
}
