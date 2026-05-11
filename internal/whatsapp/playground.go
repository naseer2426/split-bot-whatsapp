package whatsapp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const playgroundPollTrigger = "poll"

func playgroundPollOptions20() []string {
	opts := make([]string, 20)
	for i := range opts {
		opts[i] = strconv.Itoa(i + 1)
	}
	return opts
}

func (h *Handler) handlePlaygroundMode(evt *events.Message) string {
	ctx := context.Background()

	text := strings.TrimSpace(getMessageText(evt))
	fields := strings.Fields(text)

	if len(fields) >= 2 && strings.EqualFold(fields[0], "status") {
		id, err := strconv.Atoi(fields[1])
		if err != nil {
			return "Usage: status <poll_id>"
		}
		rows, err := h.GetPollStatus(ctx, id)
		if err != nil {
			return fmt.Sprintf("GetPollStatus: %v", err)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "Poll %d:\n", id)
		for _, row := range rows {
			users := "(none)"
			if len(row.Users) > 0 {
				users = strings.Join(row.Users, ", ")
			}
			fmt.Fprintf(&b, "- %s: %s\n", row.Option, users)
		}
		return strings.TrimSpace(b.String())
	}

	if strings.EqualFold(text, playgroundPollTrigger) {
		if evt.Info.Chat.Server != types.GroupServer {
			return "Playground poll: send \"poll\" from a group chat."
		}
		opts := playgroundPollOptions20()
		pollRow, err := h.SendPoll(ctx, "Playground poll", opts, evt.Info.Chat.User)
		if err != nil {
			return fmt.Sprintf("SendPoll failed: %v", err)
		}
		return fmt.Sprintf(
			"Poll sent (%d options across split WhatsApp polls). Collective poll id: %d",
			len(opts),
			pollRow.ID,
		)
	}

	return text
}
