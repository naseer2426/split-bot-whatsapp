package whatsapp

import (
	"context"
	"fmt"

	"github.com/naseer2426/split-bot-whatsapp/internal/nanobot"
	"go.mau.fi/whatsmeow/types/events"
)

func (h *Handler) handleNanobotMode(evt *events.Message) string {
	messageText := getMessageText(evt)
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
