package whatsapp

import (
	"fmt"

	"go.mau.fi/whatsmeow/types/events"
)

func (h *Handler) handleNanobotMode(evt *events.Message) string {
	return fmt.Sprintf("chat %s is in nanobot mode (not implemented), skipping...\n", evt.Info.Chat.String())
}
