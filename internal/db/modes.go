package db

import (
	"errors"
	"fmt"
)

// ChatMetaMode is the bot mode stored in whatsapp_bot_chat_meta.mode.
type ChatMetaMode string

const (
	ChatMetaModeSilent   ChatMetaMode = "silent"
	ChatMetaModeSplitBot ChatMetaMode = "splitbot"
	ChatMetaModeNanoBot  ChatMetaMode = "nanobot"
)

// ErrInvalidChatMetaMode indicates mode is not one of the ChatMetaMode* constants.
var ErrInvalidChatMetaMode = errors.New("invalid chat meta mode")

// ValidateChatMetaMode returns nil if mode is silent, splitbot, or nanobot.
func ValidateChatMetaMode(mode string) error {
	switch ChatMetaMode(mode) {
	case ChatMetaModeSilent, ChatMetaModeSplitBot, ChatMetaModeNanoBot:
		return nil
	default:
		return fmt.Errorf("%w: %q (valid: silent, splitbot, nanobot)", ErrInvalidChatMetaMode, mode)
	}
}
