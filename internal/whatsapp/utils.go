package whatsapp

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// replyToMessage carries metadata for threading an outbound message as a reply.
type replyToMessage struct {
	stanzaID     types.MessageID
	quotedSender types.JID
	chat         types.JID
}

// MessageType classifies decoded *events.Message payloads (Protobuf oneof branches on waE2E.Message).
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypePollVote MessageType = "poll_vote"
	MessageTypeUnknown  MessageType = "unknown"
)

// messageType returns a coarse Content type from evt.Message. Unknown includes nil
// message, unrecognized Protobuf branches (video, sticker, etc.), or empty payloads that do not
// match image / poll-update / plain text cues.
func messageType(evt *events.Message) MessageType {
	if evt == nil || evt.Message == nil {
		return MessageTypeUnknown
	}
	msg := evt.Message

	if msg.GetPollUpdateMessage() != nil {
		return MessageTypePollVote
	}
	if msg.GetImageMessage() != nil {
		return MessageTypeImage
	}
	if strings.TrimSpace(msg.GetConversation()) != "" || msg.GetExtendedTextMessage() != nil {
		return MessageTypeText
	}

	return MessageTypeUnknown
}

func getMessageText(evt *events.Message) string {
	if extMsg := evt.Message.GetExtendedTextMessage(); extMsg != nil {
		return extMsg.GetText()
	}
	return evt.Message.GetConversation()
}

// getPollUpdate decrypts PollUpdateMessage payloads. Returns nil when the message is not a poll vote or decryption fails.
func (h *Handler) getPollUpdate(ctx context.Context, evt *events.Message) (*waProto.PollVoteMessage, error) {
	if evt.Message.GetPollUpdateMessage() == nil {
		return nil, nil
	}
	vote, err := h.client.DecryptPollVote(ctx, evt)
	if err != nil {
		return nil, fmt.Errorf("decrypt poll vote failed: %w", err)
	}
	return vote, nil
}

// typing shows the "typing..." indicator in chat (text composition).
func (h *Handler) typing(ctx context.Context, chat types.JID) error {
	return h.client.SendChatPresence(ctx, chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
}

// stopTyping clears the typing indicator in chat.
func (h *Handler) stopTyping(ctx context.Context, chat types.JID) error {
	return h.client.SendChatPresence(ctx, chat, types.ChatPresencePaused, types.ChatPresenceMediaText)
}

// sendProcessing starts typing, and returns a function that stops typing
// and logs errors. Use as: cleanup := h.sendProcessing(ctx, evt); defer cleanup().
func (h *Handler) sendProcessing(ctx context.Context, evt *events.Message) func() {
	if err := h.typing(ctx, evt.Info.Chat); err != nil {
		fmt.Printf("Error sending typing indicator: %v\n", err)
	}
	return func() {
		if err := h.stopTyping(ctx, evt.Info.Chat); err != nil {
			fmt.Printf("Error stopping typing indicator: %v\n", err)
		}
	}
}

func (h *Handler) messageContainsBotName(message string) bool {
	return strings.Contains(strings.ToLower(message), strings.ToLower(h.botName))
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

func applyReplyContextFields(ctxInfo *waProto.ContextInfo, stanzaID types.MessageID, quotedSender, chat types.JID) {
	if stanzaID == "" {
		return
	}
	ctxInfo.StanzaID = proto.String(string(stanzaID))
	ctxInfo.RemoteJID = proto.String(chat.String())
	if chat.Server == types.GroupServer && !quotedSender.IsEmpty() {
		ctxInfo.Participant = proto.String(quotedSender.ToNonAD().String())
	}
}

func applyMentionContextFields(ctxInfo *waProto.ContextInfo, mentionedJIDs []string) {
	if len(mentionedJIDs) == 0 {
		return
	}
	ctxInfo.MentionedJID = mentionedJIDs
}

// createContextInfo builds ContextInfo from reply text (including parsed mentions) and optional thread reply metadata.
// Returns nil when there is no stanza reply and no mentions.
func createContextInfo(reply string, stanzaID types.MessageID, quotedSender, chat types.JID) *waProto.ContextInfo {
	mentionedJIDs := findMentions(reply)

	if stanzaID == "" && len(mentionedJIDs) == 0 {
		return nil
	}
	ctxInfo := &waProto.ContextInfo{}
	applyReplyContextFields(ctxInfo, stanzaID, quotedSender, chat)
	applyMentionContextFields(ctxInfo, mentionedJIDs)
	return ctxInfo
}

// composeResponse builds a text message with optional reply threading (replyTo) and mention support.
// replyTo must be nil when not replying to any message.
func composeResponse(text string, replyTo *replyToMessage) *waProto.Message {
	var stanzaID types.MessageID
	var quotedSender, chat types.JID
	if replyTo != nil {
		stanzaID = replyTo.stanzaID
		quotedSender = replyTo.quotedSender
		chat = replyTo.chat
	}

	return &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text:        proto.String(text),
			ContextInfo: createContextInfo(text, stanzaID, quotedSender, chat),
		},
	}
}
