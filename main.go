package main

import (
	"fmt"
	"log"
	"os"

	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.mau.fi/whatsmeow"

	"github.com/naseer2426/split-bot-whatsapp/internal/whatsapp"
)

var client *whatsmeow.Client

func main() {
	// Load .env file if it exists (ignore error if file doesn't exist)
	if err := godotenv.Load(); err != nil {
		// .env file is optional, so we only log if there's an actual error (not just missing file)
		if _, ok := err.(*os.PathError); !ok {
			log.Printf("Warning: Error loading .env file: %v\n", err)
		}
	}
	
	// Get DATABASE_URL from environment variables
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fmt.Println("Error: DATABASE_URL environment variable is not set")
		fmt.Println("Please set it in your .env file or as an environment variable")
		fmt.Println("Example: DATABASE_URL=postgresql://user:password@localhost:5432/whatsapp_bot?sslmode=disable")
		os.Exit(1)
	}
	
	// Initialize WhatsApp client
	var err error
	client, err = whatsapp.InitializeClient(databaseURL)
	if err != nil {
		panic(err)
	}
	
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	
	// Listen to Ctrl+C (you can also use other ways to block the main goroutine)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	

	client.Disconnect()
}
