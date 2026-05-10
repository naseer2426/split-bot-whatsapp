package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/naseer2426/split-bot-whatsapp/internal/splitbot"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// parseImage downloads and converts an image to base64.
// Returns empty ImageBase64 if imageMsg is nil.
func (h *Handler) parseImage(imageMsg *waProto.ImageMessage) (*splitbot.ImageBase64, error) {
	if imageMsg == nil {
		return nil, nil
	}

	imageBytes, err := h.client.Download(context.Background(), imageMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	base64String := base64.StdEncoding.EncodeToString(imageBytes)

	fmt.Printf("Image downloaded and converted to base64 (%d bytes, mimetype: %s)\n",
		len(imageBytes), imageMsg.GetMimetype())

	return &splitbot.ImageBase64{
		Data:  base64String,
		MType: imageMsg.GetMimetype(),
	}, nil
}

// handleSplitbotMsg runs the splitbot AI path for messages that mention the bot (or carry an image).
func (h *Handler) handleSplitbotMsg(evt *events.Message) string {
	messageText := getMessageText(evt)

	if !h.shouldProcessSplitbotMsg(
		messageText,
		evt.Message.GetImageMessage(),
	) {
		fmt.Printf("Message from %s doesn't include bot_name '%s', skipping...\n", evt.Info.Sender, h.botName)
		return ""
	}

	cleanup := h.sendProcessing(context.Background(), evt)
	defer cleanup()

	fmt.Printf("Received message from %s: %s\n", evt.Info.Sender, messageText)

	imageBase64, err := h.parseImage(evt.Message.GetImageMessage())
	if err != nil {
		fmt.Printf("Error parsing image: %v\n", err)
		return fmt.Sprintf("Error: %v", err)
	}

	req := splitbot.ProcessMessageRequest{
		Message:     messageText,
		GroupID:     evt.Info.Chat.String(),
		Sender:      cleanSenderID(evt.Info.Sender.String()),
		ImageBase64: imageBase64,
		BotName:     h.botName,
	}

	response, err := splitbot.ProcessMessage(req)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return response.Response
}

func (h *Handler) shouldProcessSplitbotMsg(
	messageText string,
	imageMsg *waProto.ImageMessage,
) bool {
	if h.botName == "" {
		return true
	}

	if imageMsg != nil {
		return true
	}

	return h.messageContainsBotName(messageText)
}
