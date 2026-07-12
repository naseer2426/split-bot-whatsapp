package whatsapp

import (
	"context"
	"fmt"

	"github.com/naseer2426/split-bot-whatsapp/internal/hermes"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (h *Handler) handleHermesMode(evt *events.Message) string {
	imageMsg := evt.Message.GetImageMessage()
	messageText := hermesMessageText(evt)

	if messageText == "" && imageMsg == nil {
		return ""
	}

	if !h.shouldProcessHermesMsg(evt, messageText, imageMsg) {
		fmt.Printf("Group message from %s doesn't include bot_name '%s' and has no image, skipping...\n", evt.Info.Sender, h.botName)
		return ""
	}

	isMention := evt.Info.Chat.Server == types.GroupServer && h.messageContainsBotName(messageText)

	if messageText != "" {
		messageText = h.nanobotMessageText(messageText)
	}
	if messageText == "" && imageMsg == nil {
		return ""
	}

	// Don't clear typing on success — Hermes replies asynchronously via egress endpoints.
	_ = h.sendProcessing(context.Background(), evt)

	req := hermes.MessageRequest{
		Sender:    evt.Info.Sender.ToNonAD().String(),
		ChatID:    evt.Info.Chat.ToNonAD().String(),
		Text:      messageText,
		MessageID: string(evt.Info.ID),
		IsMention: isMention,
	}

	if replyTo := quotedMessageID(evt); replyTo != "" {
		req.ReplyTo = replyTo
	}

	if imageMsg != nil {
		imageBase64, err := h.parseImage(imageMsg)
		if err != nil {
			fmt.Printf("Error parsing image for Hermes: %v\n", err)
			return fmt.Sprintf("Error: %v", err)
		}
		if imageBase64 != nil {
			req.Media = []hermes.MediaItem{{
				Type: "image",
				Data: imageBase64.Data,
				Mime: imageBase64.MType,
			}}
		}
	}

	if err := hermes.SendMessage(req); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return ""
}

func hermesMessageText(evt *events.Message) string {
	if text := getMessageText(evt); text != "" {
		return text
	}
	if img := evt.Message.GetImageMessage(); img != nil {
		return img.GetCaption()
	}
	return ""
}

func quotedMessageID(evt *events.Message) string {
	if evt == nil || evt.Message == nil {
		return ""
	}
	if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
		if ctx := ext.GetContextInfo(); ctx != nil {
			return ctx.GetStanzaID()
		}
	}
	if img := evt.Message.GetImageMessage(); img != nil {
		if ctx := img.GetContextInfo(); ctx != nil {
			return ctx.GetStanzaID()
		}
	}
	return ""
}

func (h *Handler) shouldProcessHermesMsg(evt *events.Message, messageText string, imageMsg *waProto.ImageMessage) bool {
	if evt.Info.Chat.Server != types.GroupServer {
		return true
	}
	if imageMsg != nil {
		return true
	}
	return h.messageContainsBotName(messageText)
}
