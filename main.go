package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"

	"github.com/naseer2426/split-bot-whatsapp/internal/config"
	"github.com/naseer2426/split-bot-whatsapp/internal/server"
	"github.com/naseer2426/split-bot-whatsapp/internal/whatsapp"
)

func main() {
	cfg := config.MustLoad()

	client, handler, err := whatsapp.InitializeWaHandler(cfg.Database.URL, cfg.Bot.Name)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Bot name: %s\n", cfg.Bot.Name)

	httpServer := server.NewServer(handler)
	httpServer.Start()

	fmt.Println("Bot is now running. Press CTRL-C to exit.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
