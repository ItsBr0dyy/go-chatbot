package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gempir/go-twitch-irc/v4"

	"github.com/itsbr0dyy/go-chatbot/commands"
	"github.com/itsbr0dyy/go-chatbot/config"
	"github.com/itsbr0dyy/go-chatbot/db"
	"github.com/itsbr0dyy/go-chatbot/misc/loop"
	"github.com/itsbr0dyy/go-chatbot/utils"
)

const joinReportDelay = 60 * time.Second

func channelsText(n int) string {
	if n == 1 {
		return "1 channel"
	}
	return fmt.Sprintf("%d channels", n)
}

func main() {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config.json"
	}

	cfg, err := config.Load(path)
	if err != nil {
		log.Fatal(err)
	}

	loop.StartEditorAccepter(cfg)

	pool, err := db.Connect(context.Background(), cfg.DatabaseURL())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	channels, err := config.LoadChannels(pool, cfg.Username)
	if err != nil {
		log.Fatalf("loading channels: %v", err)
	}

	helpers, err := config.LoadHelpers(pool)
	if err != nil {
		log.Fatalf("loading helpers: %v", err)
	}

	mods, err := config.LoadMods(pool)
	if err != nil {
		log.Fatalf("loading mods: %v", err)
	}

	reminders, err := config.LoadReminders(pool)
	if err != nil {
		log.Fatalf("loading reminders: %v", err)
	}

	deps := &commands.Deps{
		Config:    cfg,
		Channels:  channels,
		Links:     config.LoadLinks(pool),
		Helpers:   helpers,
		Mods:      mods,
		Reminders: reminders,
		Cats:      config.LoadCats(pool),
	}

	client := utils.NewClient(cfg.Username, cfg.OAuth)
	irc := client.IRC()

	var (
		startReminders sync.Once
		announce       sync.Once
		joined         atomic.Int32
		expected       atomic.Int32
	)

	sayStartup := func(text string) {
		announce.Do(func() {
			if err := client.SayReply(cfg.StartupChannel, "", text); err != nil {
				log.Printf("startup message failed: %v", err)
			}
		})
	}

	irc.OnConnect(func() {
		log.Println("connected to Twitch")
		startReminders.Do(func() { commands.StartReminderLoop(client, reminders) })

		list, err := channels.List()
		if err != nil {
			log.Printf("listing channels: %v", err)
			return
		}
		joined.Store(0)
		expected.Store(int32(len(list)))

		if len(list) == 0 {
			log.Println("joined 0 channels")
			return
		}

		time.AfterFunc(joinReportDelay, func() {
			if n, want := joined.Load(), expected.Load(); n < want {
				text := fmt.Sprintf("joined %d of %s", n, channelsText(int(want)))
				log.Println(text)
				sayStartup(text)
			}
		})
	})

	irc.OnSelfJoinMessage(func(_ twitch.UserJoinMessage) {
		if n := joined.Add(1); n == expected.Load() {
			text := fmt.Sprintf("joined %s", channelsText(int(n)))
			log.Println(text)
			sayStartup(text)
		}
	})

	irc.OnNoticeMessage(func(m twitch.NoticeMessage) {
		log.Printf("NOTICE #%s: %s", m.Channel, m.Message)
	})

	irc.OnPrivateMessage(func(m twitch.PrivateMessage) {
		msg := utils.NewMessage(m)
		go commands.DeliverReminders(client, msg, reminders)
		go commands.Handle(client, msg, deps)
	})

	list, err := channels.List()
	if err != nil {
		log.Fatalf("listing channels: %v", err)
	}
	for _, ch := range list {
		client.Join(ch)
	}
	log.Printf("connecting as %s, joining %s", cfg.Username, channelsText(len(list)))

	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}
}
