package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/naseer2426/split-bot-whatsapp/internal/splitbot"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

var client *whatsmeow.Client

// MessageType represents the type of WhatsApp message
type MessageType int

const (
	// MessageTypePrivate represents a private/direct message
	MessageTypePrivate MessageType = iota
	// MessageTypeGroup represents a group message
	MessageTypeGroup
	// MessageTypeUnknown represents an unknown or unsupported message type
	MessageTypeUnknown
)

// SetClient sets the global client for the event handler
func SetClient(c *whatsmeow.Client) {
	client = c
}

// getMessageType determines the type of message from the event
func getMessageType(evt *events.Message) MessageType {
	if evt.Message.GetExtendedTextMessage() != nil {
		return MessageTypeGroup
	}
	if evt.Message.GetConversation() != "" || evt.Message.GetImageMessage() != nil {
		return MessageTypePrivate
	}
	return MessageTypeUnknown
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

// parseImage downloads and converts an image to base64
// Returns empty ImageBase64 if imageMsg is nil
func parseImage(imageMsg *waProto.ImageMessage) (*splitbot.ImageBase64, error) {
	// Handle nil case
	if imageMsg == nil {
		return nil, nil
	}
	
	// Download the image
	imageBytes, err := client.Download(context.Background(), imageMsg)
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

// handlePrivateMessage processes a private message and returns the response
func handlePrivateMessage(evt *events.Message) *waProto.Message {
	messageText := evt.Message.GetConversation()
	fmt.Printf("Received private message from %s: %s\n", evt.Info.Sender, messageText)
	
	// Parse image (handles nil case internally)
	imageBase64, err := parseImage(evt.Message.GetImageMessage())
	if err != nil {
		fmt.Printf("Error parsing image: %v\n", err)
		return &waProto.Message{
			Conversation: proto.String(fmt.Sprintf("Error: %v", err)),
		}
	}
	
	// Build the request
	req := splitbot.ProcessMessageRequest{
		Message:    messageText,
		GroupID:    evt.Info.Chat.String(),
		Sender:     cleanSenderID(evt.Info.Sender.String()),
		ImageBase64: imageBase64,
	}
	
	// Process message with AI
	response, err := splitbot.ProcessMessage(req)
	
	var replyText string
	if err != nil {
		replyText = fmt.Sprintf("Error: %v", err)
	} else {
		replyText = response.Response
	}
	
	return &waProto.Message{
		Conversation: proto.String(replyText),
	}
}

// handleGroupMessage processes a group message and returns the response
func handleGroupMessage(evt *events.Message) *waProto.Message {
	messageText := evt.Message.GetExtendedTextMessage().GetText()
	fmt.Printf("Received group message from %s: %s\n", evt.Info.Sender, messageText)
	
	// Parse image (handles nil case internally)
	imageBase64, err := parseImage(evt.Message.GetImageMessage())
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
	}
	
	// Process message with AI
	response, err := splitbot.ProcessMessage(req)
	
	var replyText string
	if err != nil {
		replyText = fmt.Sprintf("Error: %v", err)
	} else {
		replyText = response.Response
	}
	
	return &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(replyText),
		},
	}
}

// EventHandler handles incoming WhatsApp events
func EventHandler(rawEvt interface{}) {
	switch evt := rawEvt.(type) {
	case *events.Message:
		messageType := getMessageType(evt)
		var replyMessage *waProto.Message
		
		switch messageType {
		case MessageTypeGroup:
			_, err := client.SendMessage(context.Background(), evt.Info.Chat, &waProto.Message{
				ExtendedTextMessage: &waProto.ExtendedTextMessage{
					Text: proto.String("Give me a bit..."),
				},
			})
			if err != nil {
				fmt.Printf("Error sending message: %v\n", err)
			}
			replyMessage = handleGroupMessage(evt)
		case MessageTypePrivate:
			_, err := client.SendMessage(context.Background(), evt.Info.Chat, &waProto.Message{
				Conversation: proto.String("Give me a bit..."),
			})
			if err != nil {
				fmt.Printf("Error sending message: %v\n", err)
			}
			replyMessage = handlePrivateMessage(evt)
		case MessageTypeUnknown:
			// Unknown message type, ignore
			return
		}
		
		// Send the response
		_, err := client.SendMessage(
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
