package commands

import (
	"fmt"
	"regexp"
	"strings"
)

var validChannel = regexp.MustCompile(`^[a-z0-9_]{1,25}$`)

func init() {
	Register(&Command{
		Name:     "join",
		Category: "Bot management",
		Usage:    "join <channel>",
		Desc:     "Makes the bot join a channel and saves it.",
		Access:   AccessHelper,
		Run:      join,
	})
}

func join(ctx *Context) {
	if !ctx.IsHelper() {
		return
	}

	if len(ctx.Args) == 0 {
		ctx.Reply(fmt.Sprintf("Usage: %sjoin <channel>", ctx.Config.Prefix))
		return
	}

	channel := strings.ToLower(strings.TrimPrefix(ctx.Args[0], "#"))
	if !validChannel.MatchString(channel) {
		ctx.Reply("That doesn't look like a valid channel name.")
		return
	}

	added, err := ctx.Channels.Add(channel)
	if err != nil {
		ctx.Reply(fmt.Sprintf("Couldn't save channel: %v", err))
		return
	}
	if !added {
		ctx.Reply(fmt.Sprintf("Already in %s", channel))
		return
	}

	if err := ctx.Client.Join(channel); err != nil {
		ctx.Reply(fmt.Sprintf("Saved %s but the join failed: %v", channel, err))
		return
	}
	ctx.Reply(fmt.Sprintf("Joined %s SeemsGood", channel))
}
