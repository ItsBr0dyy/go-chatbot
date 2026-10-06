package commands

import (
	"errors"
	"fmt"

	utils "github.com/itsbr0dyy/go-chatbot/utils/api"
)

func init() {
	Register(&Command{
		Name:     "subage",
		Aliases:  []string{"sa"},
		Category: "Twitch info",
		Usage:    "subage [username] [channel]",
		Desc:     "Shows a user's subscription to a channel.",
		Run:      subAge,
	})
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func subAge(ctx *Context) {
	user := ctx.Msg.User
	channel := ctx.Msg.Channel
	if len(ctx.Args) > 0 {
		user = cleanName(ctx.Args[0])
	}
	if len(ctx.Args) > 1 {
		channel = cleanName(ctx.Args[1])
	}

	if !validLogName.MatchString(user) || !validLogName.MatchString(channel) {
		ctx.Reply(fmt.Sprintf("Usage: %ssa [username] [channel]", ctx.Config.Prefix))
		return
	}

	s, err := utils.IVRSubAge(user, channel)
	if err != nil {
		if errors.Is(err, utils.ErrSubAgeNotFound) {
			ctx.Reply("Couldn't find that user or channel.")
		} else {
			ctx.Reply(fmt.Sprintf("Couldn't look that up: %v", err))
		}
		return
	}

	userName := s.User.DisplayName
	if userName == "" {
		userName = user
	}
	channelName := s.Channel.DisplayName
	if channelName == "" {
		channelName = channel
	}

	if s.IsHidden() {
		ctx.Reply(fmt.Sprintf("%s has their subscription status hidden for %s.", userName, channelName))
		return
	}

	months := s.Months()

	if !s.Subscribed() {
		if months == 0 {
			ctx.Reply(fmt.Sprintf("%s is currently not subscribed to %s, and never has been before.", userName, channelName))
		} else {
			ctx.Reply(fmt.Sprintf("%s is currently not subscribed to %s, but was previously subscribed for %s.",
				userName, channelName, plural(months, "month")))
		}
		return
	}

	kind := s.SubType()
	if kind != "sub" {
		kind += " sub"
	}
	desc := "a " + kind
	if gifter, gifted := s.Gifter(); gifted {
		desc = "a " + s.SubType() + " gifted sub"
		if s.SubType() == "sub" {
			desc = "a gifted sub"
		}
		if gifter != "" {
			desc += " by " + gifter
		}
	}

	msg := fmt.Sprintf("%s is currently subscribed to %s with %s.", userName, channelName, desc)

	if months > 0 {
		msg += fmt.Sprintf(" They have been subscribed for %s.", plural(months, "month"))
	}

	if left, renews, ok := s.Remaining(); ok && left > 0 {
		verb := "expires"
		if renews {
			verb = "renews"
		}
		msg += fmt.Sprintf(" This sub %s in %s.", verb, utils.FormatLong(left))
	}

	ctx.Reply(msg)
}
