package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"

	"github.com/naseer2426/split-bot-whatsapp/internal/config"
	"github.com/naseer2426/split-bot-whatsapp/internal/db"
	"github.com/naseer2426/split-bot-whatsapp/internal/server"
	"github.com/naseer2426/split-bot-whatsapp/internal/whatsapp"
)

func main() {
	config.MustLoad()

	dbConn, err := db.Init(config.Get().Database.URL)
	if err != nil {
		panic(err)
	}
	if err := db.RunMigrations(dbConn); err != nil {
		panic(err)
	}

	handler, err := whatsapp.NewHandler()
	if err != nil {
		panic(err)
	}
	if err := handler.Connect(context.Background()); err != nil {
		panic(err)
	}

	fmt.Printf("Bot name: %s\n", config.Get().Bot.Name)

	httpServer := server.NewServer(handler)
	httpServer.Start()

	fmt.Println("Bot is now running. Press CTRL-C to exit.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	handler.Disconnect()
}
