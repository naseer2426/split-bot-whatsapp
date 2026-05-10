package db

import (
	"time"
)

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
