package whatsapp

import (
	"regexp"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

func getMessageText(evt *events.Message) string {
	if extMsg := evt.Message.GetExtendedTextMessage(); extMsg != nil {
		return extMsg.GetText()
	}
	return evt.Message.GetConversation()
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
