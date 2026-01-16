package splitbot

// User represents a user from the API response
type User struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	TelegramUsername *string   `json:"telegram_username"`
	WhatsappNumber   *string   `json:"whatsapp_number"`
	WhatsappLID      *string   `json:"whatsapp_lid"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
}
