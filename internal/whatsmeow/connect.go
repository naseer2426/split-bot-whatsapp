package whatsmeow

import (
	"context"
	"fmt"
	"os"

	"github.com/mdp/qrterminal/v3"
	wa "go.mau.fi/whatsmeow"
)

// Connect establishes the WhatsApp session (interactive QR login when no stored session).
func Connect(ctx context.Context, client *wa.Client) error {
	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(ctx)
		if err := client.Connect(); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("\nScan this QR code with your WhatsApp app:")
				fmt.Println("1. Open WhatsApp on your phone")
				fmt.Println("2. Tap Menu (⋮) or Settings")
				fmt.Println("3. Tap Linked Devices")
				fmt.Println("4. Tap Link a Device")
				fmt.Println("5. Point your phone at this screen to scan the code")
				fmt.Println("\nQR Code:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
		return nil
	}

	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	return nil
}
