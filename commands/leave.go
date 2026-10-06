package commands

import (
	"fmt"
	"strings"
)

func init() {
	Register(&Command{
		Name:    "leave",
		Aliases: []string{"part"},
		Run:     leave,
	})
}

func leave(ctx *Context) {
	if !ctx.IsHelper() {
		return
	}

	channel := ctx.Msg.Channel
	if len(ctx.Args) > 0 {
		channel = strings.ToLower(strings.TrimPrefix(ctx.Args[0], "#"))
	}

	if channel == strings.ToLower(ctx.Config.Username) {
		ctx.Reply("I can't leave my own channel.")
		return
	}

	removed, err := ctx.Channels.Remove(channel)
	if err != nil {
		ctx.Reply(fmt.Sprintf("Couldn't update channels: %v", err))
		return
	}
	if !removed {
		ctx.Reply(fmt.Sprintf("I'm not in %s", channel))
		return
	}

	ctx.Reply(fmt.Sprintf("Leaving %s", channel))
	if err := ctx.Client.Part(channel); err != nil {
		ctx.Reply(fmt.Sprintf("Removed %s but the part failed: %v", channel, err))
	}
}
