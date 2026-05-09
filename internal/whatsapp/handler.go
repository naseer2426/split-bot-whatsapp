package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/naseer2426/split-bot-whatsapp/internal/splitbot"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// MsgHandler contains the WhatsApp client and bot configuration
type MsgHandler struct {
	client  *whatsmeow.Client
	botName string
	admins  map[string]bool
}

// NewMsgHandler creates a new Handler instance
func NewMsgHandler(client *whatsmeow.Client, botName string) *MsgHandler {
	admins, err := LoadAdmins()
	if err != nil {
		fmt.Printf("Warning: Failed to load admins: %v\n", err)
		admins = make(map[string]bool)
	}

	return &MsgHandler{
		client:  client,
		botName: botName,
		admins:  admins,
	}
}

// cleanSenderID removes @lid suffix or any suffix after : (including the :)
func cleanSenderID(sender string) string {
	// Remove anything after and including ":"
	if idx := strings.Index(sender, ":"); idx != -1 {
		return sender[:idx]
	}
	// If ":" doesn't exist, remove "@lid" suffix if present
	if strings.HasSuffix(sender, "@lid") {
		return strings.TrimSuffix(sender, "@lid")
	}
	return sender
}

// findMentions extracts all "@numbers" patterns from the response text
// and returns them as a slice of number strings
func findMentions(responseText string) []string {
	// Pattern to match @ followed by one or more digits
	pattern := regexp.MustCompile(`@(\d+)`)
	matches := pattern.FindAllStringSubmatch(responseText, -1)

	mentionedJIDs := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			lidRaw := match[1]
			// Avoid duplicates
			if !seen[lidRaw] {
				lid := lidRaw + "@lid"
				mentionedJIDs = append(mentionedJIDs, lid)
				seen[lidRaw] = true
			}
		}
	}

	return mentionedJIDs
}

// parseImage downloads and converts an image to base64
// Returns empty ImageBase64 if imageMsg is nil
func (h *MsgHandler) parseImage(imageMsg *waProto.ImageMessage) (*splitbot.ImageBase64, error) {
	// Handle nil case
	if imageMsg == nil {
		return nil, nil
	}

	// Download the image
	imageBytes, err := h.client.Download(context.Background(), imageMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	// Convert to base64
	base64String := base64.StdEncoding.EncodeToString(imageBytes)

	fmt.Printf("Image downloaded and converted to base64 (%d bytes, mimetype: %s)\n",
		len(imageBytes), imageMsg.GetMimetype())

	return &splitbot.ImageBase64{
		Data:  base64String,
		MType: imageMsg.GetMimetype(),
	}, nil
}

// handleMessage processes a message and returns the response
func (h *MsgHandler) handleMessage(evt *events.Message) *waProto.Message {
	// Get messageText: check ExtendedTextMessage first, then Conversation
	var messageText string
	if extMsg := evt.Message.GetExtendedTextMessage(); extMsg != nil {
		messageText = extMsg.GetText()
	} else {
		messageText = evt.Message.GetConversation()
	}

	fmt.Printf("Received message from %s: %s\n", evt.Info.Sender, messageText)

	// Parse image (handles nil case internally)
	imageBase64, err := h.parseImage(evt.Message.GetImageMessage())
	if err != nil {
		fmt.Printf("Error parsing image: %v\n", err)
		return &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(fmt.Sprintf("Error: %v", err)),
			},
		}
	}

	// Build the request
	req := splitbot.ProcessMessageRequest{
		Message:     messageText,
		GroupID:     evt.Info.Chat.String(),
		Sender:      cleanSenderID(evt.Info.Sender.String()),
		ImageBase64: imageBase64,
		BotName:     h.botName,
	}

	// Process message with AI
	response, err := splitbot.ProcessMessage(req)

	var replyText string
	if err != nil {
		replyText = fmt.Sprintf("Error: %v", err)
	} else {
		replyText = response.Response
	}

	return h.createTextMessage(replyText)
}

// createTextMessage creates a waProto.Message from text with mention support
func (h *MsgHandler) createTextMessage(text string) *waProto.Message {
	// Parse mentions from the text
	mentionedJIDs := findMentions(text)
	fmt.Printf("Found mentions: %v\n", mentionedJIDs)

	extendedTextMsg := &waProto.ExtendedTextMessage{
		Text: proto.String(text),
	}

	// Set ContextInfo with MentionedJID if there are any mentions
	if len(mentionedJIDs) > 0 {
		extendedTextMsg.ContextInfo = &waProto.ContextInfo{
			MentionedJID: mentionedJIDs,
		}
	}

	return &waProto.Message{
		ExtendedTextMessage: extendedTextMsg,
	}
}

func (h *MsgHandler) shouldProcessMessage(messageText string, imageMsg *waProto.ImageMessage) bool {
	if h.botName == "" {
		return true
	}

	if imageMsg != nil {
		return true
	}

	return strings.Contains(strings.ToLower(messageText), strings.ToLower(h.botName))
}

// SendMessageToGroup sends a message to a WhatsApp group
func (h *MsgHandler) SendMessageToGroup(message string, groupId string) error {
	// Create JID with groupId as User and "g.us" as Server (WhatsApp group format)
	jid := types.NewJID(groupId, types.GroupServer)

	// Create the message with mention support
	msg := h.createTextMessage(message)

	// Send the message
	_, err := h.client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return fmt.Errorf("failed to send message to group %s: %w", groupId, err)
	}

	fmt.Printf("Sent message to group %s\n", groupId)
	return nil
}

// SendMessageToUser sends a message to a WhatsApp user (1:1 chat).
// userId is either a phone number (digits, no +) or a full JID (e.g. 123@s.whatsapp.net, or ...@lid).
func (h *MsgHandler) SendMessageToUser(message string, userId string) error {
	var jid types.JID
	var err error
	if strings.Contains(userId, "@") {
		jid, err = types.ParseJID(userId)
		if err != nil {
			return fmt.Errorf("invalid user JID %q: %w", userId, err)
		}
	} else {
		jid = types.NewJID(userId, types.DefaultUserServer)
	}

	msg := h.createTextMessage(message)
	_, err = h.client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return fmt.Errorf("failed to send message to user %s: %w", userId, err)
	}

	fmt.Printf("Sent message to user %s\n", userId)
	return nil
}

// EventHandler handles incoming WhatsApp events
func (h *MsgHandler) EventHandler(rawEvt interface{}) {
	switch evt := rawEvt.(type) {
	case *events.Message:
		// Get messageText: check ExtendedTextMessage first, then Conversation
		messageText := getMessageText(evt)

		if h.isAdminMessage(evt) {
			h.handleAdminMessage(evt)
			return
		}

		// Check if message includes bot_name (case-insensitive)
		// Skip processing if bot_name is empty or message doesn't contain it
		if !h.shouldProcessMessage(messageText, evt.Message.GetImageMessage()) {
			fmt.Printf("Message from %s doesn't include bot_name '%s', skipping...\n", evt.Info.Sender, h.botName)
			return
		}

		// Send "Give me a bit..." message
		_, err := h.client.SendMessage(context.Background(), evt.Info.Chat, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String("Give me a bit..."),
			},
		})
		if err != nil {
			fmt.Printf("Error sending message: %v\n", err)
		}

		// Process the message
		replyMessage := h.handleMessage(evt)

		// Send the response
		_, err = h.client.SendMessage(
			context.Background(),
			evt.Info.Chat,
			replyMessage,
		)

		if err != nil {
			fmt.Printf("Error sending message: %v\n", err)
		} else {
			fmt.Printf("Sent response to %s\n", evt.Info.Chat)
		}
	}
}
