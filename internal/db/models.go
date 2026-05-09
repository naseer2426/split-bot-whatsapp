package db

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

const PlatformWhatsApp = "WHATSAPP"

// WhatsappBotChatMeta matches migrations through 000004_whatsapp_bot_chat_meta_mode.*.
type WhatsappBotChatMeta struct {
	ID           int       `gorm:"column:id;primaryKey;autoIncrement"`
	GroupID      string    `gorm:"column:group_id;type:varchar;not null;uniqueIndex"`
	PlatformType string    `gorm:"column:platform_type;type:varchar;not null"`
	Mode         string    `gorm:"column:mode;type:varchar;not null;default:silent"`
	CreatedAt    time.Time `gorm:"column:created_at;type:timestamptz"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:timestamptz"`
}

func (WhatsappBotChatMeta) TableName() string {
	return "whatsapp_bot_chat_meta"
}

// IsChatWhitelisted reports whether group_id matches a row in whatsapp_bot_chat_meta.
func IsChatWhitelisted(db *gorm.DB, chatID string) (bool, error) {
	var count int64
	err := db.Model(&WhatsappBotChatMeta{}).Where("group_id = ?", chatID).Limit(1).Count(&count).Error
	return count > 0, err
}

// UpsertWhitelistedChat inserts whatsapp_bot_chat_meta for chatID or updates mode and platform_type if it already exists.
func UpsertWhitelistedChat(db *gorm.DB, chatID, mode string) error {
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
	var row WhatsappBotChatMeta
	if err := db.Where("group_id = ?", chatID).First(&row).Error; err != nil {
		return err
	}
	row.Mode = mode
	row.PlatformType = PlatformWhatsApp
	return db.Save(&row).Error
}
