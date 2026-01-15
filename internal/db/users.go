package db

import (
	"time"
)

// SplitBotUser represents the split_bot_users table
type SplitBotUser struct {
	ID              uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"type:varchar(255);not null" json:"name"`
	Email           string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_email" json:"email"`
	TelegramUsername *string       `gorm:"type:varchar(255);index:idx_telegram_username" json:"telegram_username"`
	WhatsappNumber  *string        `gorm:"type:varchar(50)" json:"whatsapp_number"`
	WhatsappLID     *string        `gorm:"type:varchar(50)" json:"whatsapp_lid"`
	CreatedAt       time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (SplitBotUser) TableName() string {
	return "split_bot_users"
}

// GetAllUsers retrieves all users from the split_bot_users table
func GetAllUsers() ([]SplitBotUser, error) {
	var users []SplitBotUser
	db := GetDB()
	
	if err := db.Find(&users).Error; err != nil {
		return nil, err
	}
	
	return users, nil
}

// UpdateUsers updates multiple users in the split_bot_users table
// Each user must have a valid ID to be updated
func UpdateUsers(users []SplitBotUser) error {
	db := GetDB()
	
	for _, user := range users {
		if user.ID == 0 {
			continue // Skip users without valid ID
		}
		
		if err := db.Save(&user).Error; err != nil {
			return err
		}
	}
	
	return nil
}
