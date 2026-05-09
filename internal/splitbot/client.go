package splitbot

import (
	"time"

	"github.com/naseer2426/split-bot-whatsapp/internal/config"
	"resty.dev/v3"
)

// getClient returns a configured resty client with base URL and logging.
func getClient() *resty.Client {
	domain := config.Get().SplitBot.URL
	client := resty.New().
		SetBaseURL(domain).
		SetTimeout(120 * time.Second).
		SetDebug(true).
		SetDebugBodyLimit(10 * 1024) // Limit body logs to 10KB

	return client
}
