package hermes

import (
	"time"

	"github.com/naseer2426/split-bot-whatsapp/internal/config"
	"resty.dev/v3"
)

func getClient() *resty.Client {
	cfg := config.Get().Hermes
	client := resty.New().
		SetBaseURL(cfg.URL).
		SetTimeout(120 * time.Second).
		SetDebug(true).
		SetDebugBodyLimit(10 * 1024)

	if cfg.APIKey != "" {
		client.SetAuthToken(cfg.APIKey)
	}

	return client
}
