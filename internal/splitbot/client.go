package splitbot

import (
	"fmt"
	"os"
	"time"

	"resty.dev/v3"
)

// getClient returns a configured resty client with base URL and logging
func getClient() (*resty.Client, error) {
	domain := os.Getenv("SPLIT_BOT_URL")
	if domain == "" {
		return nil, fmt.Errorf("SPLIT_BOT_URL environment variable is not set")
	}

	client := resty.New().
		SetBaseURL(domain).
		SetTimeout(120 * time.Second).
		SetDebug(true).
		SetDebugBodyLimit(10 * 1024) // Limit body logs to 10KB

	return client, nil
}
