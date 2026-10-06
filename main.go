package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/gempir/go-twitch-irc/v4"

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

	client := utils.NewClient(cfg.Username, cfg.OAuth)
	irc := client.IRC()

	irc.OnConnect(func() {
		log.Println("connected to Twitch")
	})

	irc.OnNoticeMessage(func(m twitch.NoticeMessage) {
		log.Printf("NOTICE #%s: %s", m.Channel, m.Message)
	})

	irc.OnPrivateMessage(func(m twitch.PrivateMessage) {
		msg := &utils.Message{
			Tags:    m.Tags,
			User:    m.User.Name,
			Channel: m.Channel,
			Text:    m.Message,
		}
		go commands.Handle(client, msg, cfg, channels, links, helpers)
	})

	list, err := channels.List()
	if err != nil {
		log.Fatalf("listing channels: %v", err)
	}
	for _, ch := range list {
		client.Join(ch)
	}
	log.Printf("connecting as %s, joining %s", cfg.Username, strings.Join(list, ", "))

	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}
}
