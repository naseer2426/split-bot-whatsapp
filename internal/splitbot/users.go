package splitbot

import (
	"fmt"
)

// GetAllUsersOptions represents query parameters for GetAllUsers
type GetAllUsersOptions struct {
	Limit  int
	Offset int
}

// CreateUserRequest represents the payload for creating a new user
type CreateUserRequest struct {
	Name             string  `json:"name"`
	Email            string  `json:"email"`
	TelegramUsername *string `json:"telegram_username,omitempty"`
	WhatsappNumber   *string `json:"whatsapp_number,omitempty"`
	WhatsappLID      *string `json:"whatsapp_lid,omitempty"`
}

// GetAllUsers retrieves all users from the API
// It uses the SPLIT_BOT_URL environment variable to determine the API base URL
func GetAllUsers(opts GetAllUsersOptions) ([]User, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}

	// Build query parameters
	queryParams := make(map[string]string)
	if opts.Limit > 0 {
		queryParams["limit"] = fmt.Sprintf("%d", opts.Limit)
	}
	if opts.Offset > 0 {
		queryParams["offset"] = fmt.Sprintf("%d", opts.Offset)
	}

	var users []User
	req := client.R().
		SetResult(&users)

	if len(queryParams) > 0 {
		req.SetQueryParams(queryParams)
	}

	resp, err := req.Get("/users")
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("API returned non-success status: %d, body: %s", resp.StatusCode(), resp.String())
	}

	return users, nil
}

// CreateUser creates a new user via the API
// It uses the SPLIT_BOT_URL environment variable to determine the API base URL
func CreateUser(req CreateUserRequest) (*User, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}

	var user User
	resp, err := client.R().
		SetBody(req).
		SetResult(&user).
		Post("/users")

	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("API returned non-success status: %d, body: %s", resp.StatusCode(), resp.String())
	}

	return &user, nil
}

// UpdateUser updates an existing user via the API
// It uses the SPLIT_BOT_URL environment variable to determine the API base URL
func UpdateUser(userID int, req CreateUserRequest) (*User, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}

	var user User
	resp, err := client.R().
		SetBody(req).
		SetResult(&user).
		Put(fmt.Sprintf("/users/%d", userID))

	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("API returned non-success status: %d, body: %s", resp.StatusCode(), resp.String())
	}

	return &user, nil
}
