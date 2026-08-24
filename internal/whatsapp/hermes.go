package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/naseer2426/split-bot-whatsapp/internal/hermes"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (h *Handler) handleHermesMode(evt *events.Message) string {
	messageText := hermesMessageText(evt)
	hasMedia := hasHermesInboundMedia(evt)

	if messageText == "" && !hasMedia {
		return ""
	}

	if !h.shouldProcessHermesMsg(evt, messageText, hasMedia) {
		fmt.Printf("Group message from %s doesn't include bot_name '%s' and has no media, skipping...\n", evt.Info.Sender, h.botName)
		return ""
	}

	isMention := evt.Info.Chat.Server == types.GroupServer && h.messageContainsBotName(messageText)

	if messageText != "" {
		messageText = h.nanobotMessageText(messageText)
	}

	media, err := h.hermesMediaItems(evt)
	if err != nil {
		fmt.Printf("Error parsing media for Hermes: %v\n", err)
		return fmt.Sprintf("Error: %v", err)
	}

	if messageText == "" && len(media) == 0 {
		return ""
	}

	// Don't clear typing on success — Hermes replies asynchronously via egress endpoints.
	_ = h.sendProcessing(context.Background(), evt)

	req := hermes.MessageRequest{
		Sender:    evt.Info.Sender.ToNonAD().String(),
		ChatID:    evt.Info.Chat.ToNonAD().String(),
		Text:      messageText,
		Media:     media,
		MessageID: string(evt.Info.ID),
		IsMention: isMention,
	}

	if replyTo := quotedMessageID(evt); replyTo != "" {
		req.ReplyTo = replyTo
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
	if evt == nil || evt.Message == nil {
		return ""
	}
	if img := evt.Message.GetImageMessage(); img != nil {
		if caption := img.GetCaption(); caption != "" {
			return caption
		}
	}
	if doc := evt.Message.GetDocumentMessage(); doc != nil {
		return doc.GetCaption()
	}
	return ""
}

func hasHermesInboundMedia(evt *events.Message) bool {
	if evt == nil || evt.Message == nil {
		return false
	}
	return evt.Message.GetImageMessage() != nil || evt.Message.GetDocumentMessage() != nil
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
	if doc := evt.Message.GetDocumentMessage(); doc != nil {
		if ctx := doc.GetContextInfo(); ctx != nil {
			return ctx.GetStanzaID()
		}
	}
	return ""
}

func (h *Handler) shouldProcessHermesMsg(evt *events.Message, messageText string, hasMedia bool) bool {
	if evt.Info.Chat.Server != types.GroupServer {
		return true
	}
	if hasMedia {
		return true
	}
	return h.messageContainsBotName(messageText)
}

func (h *Handler) hermesMediaItems(evt *events.Message) ([]hermes.MediaItem, error) {
	if evt == nil || evt.Message == nil {
		return nil, nil
	}

	var items []hermes.MediaItem

	if img := evt.Message.GetImageMessage(); img != nil {
		item, err := h.downloadHermesMedia("image", img, img.GetMimetype(), "")
		if err != nil {
			return nil, err
		}
		if item != nil {
			items = append(items, *item)
		}
	}

	if doc := evt.Message.GetDocumentMessage(); doc != nil {
		kind := hermesMediaKind(doc.GetMimetype(), "document")
		item, err := h.downloadHermesMedia(kind, doc, doc.GetMimetype(), doc.GetFileName())
		if err != nil {
			return nil, err
		}
		if item != nil {
			items = append(items, *item)
		}
	}

	return items, nil
}

func hermesMediaKind(mime, fallback string) string {
	primary := strings.ToLower(strings.TrimSpace(strings.Split(mime, ";")[0]))
	switch {
	case strings.HasPrefix(primary, "image/"):
		return "image"
	case strings.HasPrefix(primary, "video/"):
		return "video"
	case strings.HasPrefix(primary, "audio/"):
		return "audio"
	case fallback != "":
		return fallback
	default:
		return "document"
	}
}

func (h *Handler) downloadHermesMedia(
	mediaType string,
	msg whatsmeow.DownloadableMessage,
	mime, filename string,
) (*hermes.MediaItem, error) {
	data, err := h.downloadMedia(context.Background(), msg)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", mediaType, err)
	}
	if len(data) == 0 {
		return nil, nil
	}

	fmt.Printf("Hermes media downloaded (%s, %d bytes, mimetype: %s, filename: %s)\n",
		mediaType, len(data), mime, filename)

	return &hermes.MediaItem{
		Type:     mediaType,
		Data:     base64.StdEncoding.EncodeToString(data),
		Mime:     mime,
		Filename: filename,
	}, nil
}
