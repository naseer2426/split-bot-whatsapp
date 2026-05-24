package nanobot

import (
	"time"

	"github.com/naseer2426/split-bot-whatsapp/internal/config"
	"resty.dev/v3"
)

func getClient() *resty.Client {
	return resty.New().
		SetBaseURL(config.Get().Nanobot.URL).
		SetTimeout(30 * time.Second).
		SetDebug(true).
		SetDebugBodyLimit(10 * 1024)
}
