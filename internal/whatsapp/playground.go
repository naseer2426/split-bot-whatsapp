package whatsapp

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	wa "go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

const playgroundPollTrigger = "poll"

// playgroundPollOptions must stay in sync with the poll built when playgroundPollTrigger is sent.
var playgroundPollOptions = []string{"A", "B", "C", "D", "E"}

// playgroundPollHashHex maps SHA-256(option UTF-8) hex → option label (WhatsApp poll vote encoding).
var playgroundPollHashHex map[string]string

func init() {
	hashes := wa.HashPollOptions(playgroundPollOptions)
	playgroundPollHashHex = make(map[string]string, len(hashes))
	for i, sum := range hashes {
		playgroundPollHashHex[hex.EncodeToString(sum)] = playgroundPollOptions[i]
	}
}

func formatPlaygroundPollVote(sender string, vote *waProto.PollVoteMessage) string {
	if vote == nil {
		return ""
	}
	opts := vote.GetSelectedOptions()
	if len(opts) == 0 {
		return ""
	}
	labels := make([]string, 0, len(opts))
	for _, h := range opts {
		label, ok := playgroundPollHashHex[hex.EncodeToString(h)]
		if !ok {
			label = "?"
		}
		labels = append(labels, label)
	}
	return fmt.Sprintf("%s voted for option %s", sender, strings.Join(labels, ", "))
}

func (h *Handler) handlePlaygroundMode(evt *events.Message) string {
	if evt.Message.GetPollUpdateMessage() != nil {
		vote, err := h.getPollUpdate(context.Background(), evt)
		if err != nil {
			fmt.Printf("playground poll vote: %v\n", err)
			return ""
		}
		if vote == nil {
			return ""
		}
		return formatPlaygroundPollVote(cleanSenderID(evt.Info.Sender.String()), vote)
	}

	text := strings.TrimSpace(getMessageText(evt))
	if strings.EqualFold(text, playgroundPollTrigger) {
		poll := h.client.BuildPollCreation(
			"Playground",
			playgroundPollOptions,
			len(playgroundPollOptions),
		)
		_, err := h.client.SendMessage(context.Background(), evt.Info.Chat, poll)
		if err != nil {
			return fmt.Sprintf("failed to send poll: %v", err)
		}
		return ""
	}
	return text
}
