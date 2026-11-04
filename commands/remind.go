package commands

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/itsbr0dyy/go-chatbot/config"
	"github.com/itsbr0dyy/go-chatbot/utils"
)

const (
	maxPendingReminders = 20
	maxReminderLen      = 300
	maxRemindYears      = 5
)

func init() {
	Register(&Command{
		Name:     "remindme",
		Category: "Utility",
		Usage:    "remindme <time> <message>",
		Desc:     "Reminds you after a set time. Time can be like 10m, 3d, 1h30m or 2months.",
		Run:      remindMe,
	})
	Register(&Command{
		Name:     "remind",
		Category: "Utility",
		Usage:    "remind <username> [time] <message>",
		Desc:     "Leaves a message for a user, delivered when they next chat (or after the time, if given).",
		Run:      remind,
	})
}

func parseWhen(args []string) (time.Time, int) {
	span, used := utils.ParseSpan(args)
	if used == 0 {
		return time.Time{}, 0
	}
	return span.Apply(time.Now()), used
}

func checkWhen(at time.Time) string {
	now := time.Now()
	if !at.After(now.Add(time.Second)) {
		return "That time is too short."
	}
	if at.After(now.AddDate(maxRemindYears, 0, 0)) {
		return fmt.Sprintf("I can only remind up to %d years ahead.", maxRemindYears)
	}
	return ""
}

func remindMe(ctx *Context) {
	usage := fmt.Sprintf("Usage: %sremindme <time> <message>, for example 10m, 3d or 2months", ctx.Config.Prefix)

	at, used := parseWhen(ctx.Args)
	if used == 0 || used == len(ctx.Args) {
		ctx.Reply(usage)
		return
	}
	if msg := checkWhen(at); msg != "" {
		ctx.Reply(msg)
		return
	}

	saveReminder(ctx, ctx.Msg.User, strings.Join(ctx.Args[used:], " "), &at, "I'll remind you")
}

func remind(ctx *Context) {
	usage := fmt.Sprintf("Usage: %sremind <username> [time] <message>", ctx.Config.Prefix)

	if len(ctx.Args) < 2 {
		ctx.Reply(usage)
		return
	}

	target := cleanName(ctx.Args[0])
	if !validLogName.MatchString(target) {
		ctx.Reply("That doesn't look like a valid Twitch username.")
		return
	}
	if strings.EqualFold(target, ctx.Config.Username) {
		ctx.Reply("I can't remind myself.")
		return
	}

	rest := ctx.Args[1:]
	var at *time.Time

	t, used := parseWhen(rest)
	if used > 0 && used == len(rest) {
		ctx.Reply(usage)
		return
	}
	if used > 0 {
		if msg := checkWhen(t); msg != "" {
			ctx.Reply(msg)
			return
		}
		at = &t
		rest = rest[used:]
	}

	saveReminder(ctx, target, strings.Join(rest, " "), at, "I'll remind "+target)
}

func saveReminder(ctx *Context, target, text string, at *time.Time, who string) {
	if len([]rune(text)) > maxReminderLen {
		ctx.Reply(fmt.Sprintf("That message is too long (max %d characters).", maxReminderLen))
		return
	}

	n, err := ctx.Reminders.CountPending(ctx.Msg.User)
	if err != nil {
		ctx.Reply(fmt.Sprintf("Couldn't save that: %v", err))
		return
	}
	if n >= maxPendingReminders {
		ctx.Reply(fmt.Sprintf("You already have %d pending reminders.", n))
		return
	}

	if err := ctx.Reminders.Add(target, ctx.Msg.User, ctx.Msg.Channel, text, at); err != nil {
		ctx.Reply(fmt.Sprintf("Couldn't save that: %v", err))
		return
	}

	if at != nil {
		ctx.Reply(fmt.Sprintf("Okay, %s in %s.", who, utils.FormatLong(time.Until(*at).Round(time.Second))))
	} else {
		ctx.Reply(fmt.Sprintf("Okay, %s next time they chat.", who))
	}
}

func DeliverReminders(client *utils.Client, msg *utils.Message, store *config.Reminders) {
	if !store.HasWaiting(msg.User) {
		return
	}

	list, err := store.TakeWaiting(msg.User)
	if err != nil {
		log.Printf("reminders: %v", err)
		return
	}
	for _, r := range list {
		text := fmt.Sprintf("%s left you a message (%s ago): %s", r.Author, utils.FormatAgo(r.CreatedAt), r.Message)
		client.SayReply(msg.Channel, msg.Tags["id"], text)
	}
}

func StartReminderLoop(client *utils.Client, store *config.Reminders) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			due, err := store.TakeDue()
			if err != nil {
				log.Printf("reminders: %v", err)
				continue
			}
			for _, r := range due {
				ago := utils.FormatAgo(r.CreatedAt)
				var text string
				if r.Author == r.Target {
					text = fmt.Sprintf("@%s, reminder from %s ago: %s", r.Target, ago, r.Message)
				} else {
					text = fmt.Sprintf("@%s, reminder from %s (%s ago): %s", r.Target, r.Author, ago, r.Message)
				}
				client.SayReply(r.Channel, "", text)
			}
		}
	}()
}
