package whatsapp

import (
	"context"
	"fmt"
	"os"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// InitializeWaHandler sets up and returns a WhatsApp client and handler
func InitializeWaHandler(databaseURL string, botName string) (*whatsmeow.Client, *WaHandler, error) {
	// Set up logging
	dbLog := waLog.Stdout("Database", "INFO", true)
	ctx := context.Background()

	// Initialize the PostgreSQL container for storing session data
	container, err := sqlstore.New(ctx, "postgres", databaseURL, dbLog)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create SQL store: %w", err)
	}

	// If you want multiple sessions, use container.NewDevice()
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get device store: %w", err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Create waHandler with client and bot name
	waHandler := NewWaHandler(client, botName)

	// Add event handler for incoming messages
	client.AddEventHandler(waHandler.EventHandler)

	// Connect to WhatsApp
	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect: %w", err)
		}

		// Print QR code for scanning
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("\nScan this QR code with your WhatsApp app:")
				fmt.Println("1. Open WhatsApp on your phone")
				fmt.Println("2. Tap Menu (⋮) or Settings")
				fmt.Println("3. Tap Linked Devices")
				fmt.Println("4. Tap Link a Device")
				fmt.Println("5. Point your phone at this screen to scan the code")
				fmt.Println("\nQR Code:")
				// Display QR code in terminal using ASCII art
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	return client, waHandler, nil
}
