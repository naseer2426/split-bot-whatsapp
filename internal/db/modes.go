package db

import (
	"errors"
	"fmt"
)

// ChatMetaMode is the bot mode stored in whatsapp_bot_chat_meta.mode.
type ChatMetaMode string

const (
	ChatMetaModeSilent     ChatMetaMode = "silent"
	ChatMetaModeSplitBot   ChatMetaMode = "splitbot"
	ChatMetaModeNanoBot    ChatMetaMode = "nanobot"
	ChatMetaModeHermes     ChatMetaMode = "hermes"
	ChatMetaModePlayground ChatMetaMode = "playground"
)

// ErrInvalidChatMetaMode indicates mode is not one of the ChatMetaMode* constants.
var ErrInvalidChatMetaMode = errors.New("invalid chat meta mode")

// ValidateChatMetaMode returns nil if mode is a known ChatMetaMode*.
func ValidateChatMetaMode(mode string) error {
	switch ChatMetaMode(mode) {
	case ChatMetaModeSilent, ChatMetaModeSplitBot, ChatMetaModeNanoBot, ChatMetaModeHermes, ChatMetaModePlayground:
		return nil
	default:
		return fmt.Errorf("%w: %q (valid: silent, splitbot, nanobot, hermes, playground)", ErrInvalidChatMetaMode, mode)
	}
}
