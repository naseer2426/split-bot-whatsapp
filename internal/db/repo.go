package db

import (
	"errors"

	"gorm.io/gorm"
)

const PlatformWhatsApp = "WHATSAPP"

// GetWhatsappBotChatMeta loads the full whatsapp_bot_chat_meta row for chatID.
// When no row exists, it returns (nil, false, nil).
func GetWhatsappBotChatMeta(db *gorm.DB, chatID string) (*WhatsappBotChatMeta, bool, error) {
	var row WhatsappBotChatMeta
	err := db.Where("group_id = ?", chatID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &row, true, nil
}

// GetChatMeta loads whatsapp_bot_chat_meta for chatID.
// When no row exists, it returns ("", false, nil).
// On success it returns (mode, true, nil); on DB errors it returns (_, false, err).
func GetChatMeta(db *gorm.DB, chatID string) (mode string, whitelisted bool, err error) {
	row, ok, err := GetWhatsappBotChatMeta(db, chatID)
	if err != nil || !ok {
		return "", ok, err
	}
	return row.Mode, true, nil
}

// UpsertWhitelistedChat inserts whatsapp_bot_chat_meta for chatID or updates mode and platform_type if it already exists.
func UpsertWhitelistedChat(db *gorm.DB, chatID, mode string) error {
	if err := ValidateChatMetaMode(mode); err != nil {
		return err
	}
	var row WhatsappBotChatMeta
	err := db.Where("group_id = ?", chatID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = WhatsappBotChatMeta{
			GroupID:      chatID,
			PlatformType: PlatformWhatsApp,
			Mode:         mode,
		}
		return db.Create(&row).Error
	}
	if err != nil {
		return err
	}
	row.Mode = mode
	row.PlatformType = PlatformWhatsApp
	return db.Save(&row).Error
}

// SetChatMode updates mode for an existing whatsapp_bot_chat_meta row. Returns gorm.ErrRecordNotFound if chatID is absent.
func SetChatMode(db *gorm.DB, chatID, mode string) error {
	if err := ValidateChatMetaMode(mode); err != nil {
		return err
	}
	var row WhatsappBotChatMeta
	if err := db.Where("group_id = ?", chatID).First(&row).Error; err != nil {
		return err
	}
	row.Mode = mode
	row.PlatformType = PlatformWhatsApp
	return db.Save(&row).Error
}
