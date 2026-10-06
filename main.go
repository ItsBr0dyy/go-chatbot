package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/itsbr0dyy/go-chatbot/commands"
	"github.com/itsbr0dyy/go-chatbot/config"
	"github.com/itsbr0dyy/go-chatbot/db"
	"github.com/itsbr0dyy/go-chatbot/utils"
)

func main() {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config.json"
	}

	cfg, err := config.Load(path)
	if err != nil {
		log.Fatal(err)
	}

	pool, err := db.Connect(context.Background(), cfg.DatabaseURL())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	channels, err := config.LoadChannels(pool, cfg.Username)
	if err != nil {
		log.Fatalf("loading channels: %v", err)
	}
	links := config.LoadLinks(pool)

	helpers, err := config.LoadHelpers(pool)
	if err != nil {
		log.Fatalf("loading helpers: %v", err)
	}

	for {
		if err := run(cfg, channels, links, helpers); err != nil {
			log.Printf("disconnected: %v", err)
		}
		log.Println("reconnecting in 5s...")
		time.Sleep(5 * time.Second)
	}
}

func run(cfg *config.Config, channels *config.Channels, links *config.Links, helpers *config.Helpers) error {
	client, err := utils.Dial()
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Login(cfg.Username, cfg.OAuth); err != nil {
		return err
	}

	list, err := channels.List()
	if err != nil {
		return err
	}
	for _, ch := range list {
		if err := client.Join(ch); err != nil {
			return err
		}
	}
	log.Printf("connecting as %s, joining %s", cfg.Username, strings.Join(list, ", "))

	for {
		line, err := client.ReadLine()
		if err != nil {
			return err
		}

		msg := utils.ParseMessage(line)
		switch msg.Command {
		case "001":
			log.Println("authenticated OK")
		case "PING":
			client.Send("PONG :tmi.twitch.tv")
		case "PONG":
			utils.HandlePong(msg.Text)
		case "PRIVMSG":
			go commands.Handle(client, msg, cfg, channels, links, helpers)
		case "NOTICE":
			log.Printf("NOTICE: %s", msg.Text)
			if strings.Contains(msg.Text, "authentication failed") ||
				strings.Contains(msg.Text, "improperly formatted auth") {
				log.Fatal("bad token or username")
			}
		}
	}
}
