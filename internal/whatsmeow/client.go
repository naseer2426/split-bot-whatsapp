package whatsmeow

import (
	"context"
	"fmt"

	"github.com/naseer2426/split-bot-whatsapp/internal/config"
	wa "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// InitializeWhatsmeow opens the PostgreSQL session store from config, initializes the device store,
// and returns a WhatsApp client (not yet connected).
func InitializeWhatsmeow(ctx context.Context) (*wa.Client, error) {
	deviceStore, err := getDeviceStore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device store: %w", err)
	}

	clientLogger := waLog.Stdout("Client", "INFO", true)
	return wa.NewClient(deviceStore, clientLogger), nil
}

func getDeviceStore(ctx context.Context) (*store.Device, error) {
	dbLogger := waLog.Stdout("Database", "INFO", true)

	container, err := sqlstore.New(ctx, "postgres", config.Get().Database.URL, dbLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQL store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device store: %w", err)
	}

	return deviceStore, nil
}
