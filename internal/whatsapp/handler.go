package whatsapp

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/naseer2426/split-bot-whatsapp/internal/config"
	"github.com/naseer2426/split-bot-whatsapp/internal/db"
	waclient "github.com/naseer2426/split-bot-whatsapp/internal/whatsmeow"
	wa "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// Handler contains the WhatsApp client and bot configuration
type Handler struct {
	client         *wa.Client
	botName        string
	admins         map[string]bool
	db             *gorm.DB
	handlersByMode map[string]MsgHandler
}

// MsgHandler handles an incoming WhatsApp message for one chat meta mode (splitbot, nanobot, etc.).
// Return empty string to send nothing (silent / filtered / not implemented).
type MsgHandler func(msg *events.Message) string

// NewHandler builds a handler with a WhatsApp client and registers the event handler (call Connect next).
func NewHandler() (*Handler, error) {
	client, err := waclient.InitializeWhatsmeow(context.Background())
	if err != nil {
		return nil, err
	}

	admins, err := LoadAdmins()
	if err != nil {
		fmt.Printf("Warning: Failed to load admins: %v\n", err)
		admins = make(map[string]bool)
	}

	h := &Handler{
		client:  client,
		botName: config.Get().Bot.Name,
		admins:  admins,
		db:      db.GetDB(),
	}
	h.handlersByMode = map[string]MsgHandler{
		string(db.ChatMetaModeSilent):     h.handleSilentMode,
		string(db.ChatMetaModeSplitBot):   h.handleSplitbotMsg,
		string(db.ChatMetaModeNanoBot):    h.handleNanobotMode,
		string(db.ChatMetaModePlayground): h.handlePlaygroundMode,
	}
	h.registerClient()
	return h, nil
}

func (h *Handler) registerClient() {
	h.client.AddEventHandler(h.EventHandler)
}

// Connect establishes the WhatsApp session (including QR login when needed).
func (h *Handler) Connect(ctx context.Context) error {
	return waclient.Connect(ctx, h.client)
}

// Disconnect closes the WhatsApp connection.
func (h *Handler) Disconnect() {
	h.client.Disconnect()
}

func (h *Handler) handleSilentMode(evt *events.Message) string {
	fmt.Printf("chat %s is in silent mode, skipping...\n", evt.Info.Chat.String())
	return ""
}

// handle loads chat authorization and delegates to the mode handler.
// Returned text is sent as-is; empty string means do not reply.
func (h *Handler) handle(evt *events.Message) string {
	chatID := evt.Info.Chat.String()
	mode, allowed, err := db.GetChatMeta(h.db, chatID)
	if err != nil {
		return fmt.Sprintf("chat meta lookup failed for %s: %v", chatID, err)
	}
	if !allowed {
		return h.handleNotWhitelisted(evt)
	}

	if messageType(evt) == MessageTypePollVote {
		if err := h.HandlePollVote(context.Background(), evt); err != nil {
			fmt.Printf("HandlePollVote: %v\n", err)
		}
		return ""
	}

	fn, ok := h.handlersByMode[mode]
	if !ok {
		return fmt.Sprintf("unsupported chat mode %q", mode)
	}
	return fn(evt)
}

func (h *Handler) handleNotWhitelisted(evt *events.Message) string {
	msg := getMessageText(evt)
	if !h.messageContainsBotName(msg) {
		fmt.Printf("Message from %s doesn't include bot_name '%s', skipping...\n", evt.Info.Sender, h.botName)
		return ""
	}
	return fmt.Sprintf(
		"This chat (%s) is not whitelisted in the DB, please ask Naseer to whitelist it",
		evt.Info.Chat.String(),
	)
}

// SendMessageToGroup sends a message to a WhatsApp group
func (h *Handler) SendMessageToGroup(message string, groupId string) error {
	// Create JID with groupId as User and "g.us" as Server (WhatsApp group format)
	jid := types.NewJID(groupId, types.GroupServer)

	// Create the message with mention support
	msg := composeResponse(message, nil /* replyToMessage */)

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
func (h *Handler) SendMessageToUser(message string, userId string) error {
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

	msg := composeResponse(message, nil /* replyToMessage */)
	_, err = h.client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return fmt.Errorf("failed to send message to user %s: %w", userId, err)
	}

	fmt.Printf("Sent message to user %s\n", userId)
	return nil
}

// SendMessageToChat sends a plain text message to any WhatsApp chat identified by a full JID string.
// chatId must parse with types.ParseJID (e.g. Phone@s.whatsapp.net, ...@g.us for groups).
func (h *Handler) SendMessageToChat(message string, chatId string) error {
	jid, err := types.ParseJID(chatId)
	if err != nil {
		return fmt.Errorf("invalid chat JID %q: %w", chatId, err)
	}

	msg := composeResponse(message, nil /* replyToMessage */)
	_, err = h.client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return fmt.Errorf("failed to send message to chat %s: %w", chatId, err)
	}

	fmt.Printf("Sent message to chat %s\n", chatId)
	return nil
}

// EventHandler handles incoming WhatsApp events
func (h *Handler) EventHandler(rawEvt interface{}) {
	switch evt := rawEvt.(type) {
	case *events.Message:

		if h.isAdminMessage(evt) {
			h.handleAdminMessage(evt)
			return
		}

		replyTo := &replyToMessage{
			stanzaID:     evt.Info.ID,
			quotedSender: evt.Info.Sender,
			chat:         evt.Info.Chat,
		}

		text := h.handle(evt)
		if text == "" {
			return
		}

		_, sendErr := h.client.SendMessage(
			context.Background(),
			evt.Info.Chat,
			composeResponse(text, replyTo),
		)
		if sendErr != nil {
			fmt.Printf("Error sending message: %v\n", sendErr)
		} else {
			fmt.Printf("Sent response to %s\n", evt.Info.Chat)
		}
	}
}
