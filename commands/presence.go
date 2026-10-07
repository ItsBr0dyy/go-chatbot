package commands

import (
	"errors"
	"fmt"
	"strings"

	utils "github.com/itsbr0dyy/go-chatbot/utils/api"
	"github.com/itsbr0dyy/go-chatbot/utils/api/stv"
)

func init() {
	Register(&Command{
		Name:     "presence",
		Category: "7TV",
		Usage:    "presence [username]",
		Desc:     "Sends a 7TV presence for you or another user in this channel.",
		Run:      presence,
	})
}

func presence(ctx *Context) {
	name := ctx.Msg.DisplayName()
	twitchID := ctx.Msg.UserID()

	if len(ctx.Args) > 0 {
		target := strings.ToLower(strings.TrimPrefix(ctx.Args[0], "@"))
		if !strings.EqualFold(target, ctx.Msg.User) {
			name = target
			id, err := utils.TwitchIDByLogin(target)
			if err != nil {
				ctx.Reply(fmt.Sprintf("Couldn't find Twitch user %s.", target))
				return
			}
			twitchID = id
		}
	}

	stvID, err := stv.UserIDFromTwitch(twitchID)
	if err != nil {
		if errors.Is(err, stv.ErrUserNotFound) {
			ctx.Reply(fmt.Sprintf("%s doesn't have a 7TV profile.", name))
		} else {
			ctx.Reply(fmt.Sprintf("Couldn't reach 7TV API: %v", err))
		}
		return
	}

	if err := stv.SendPresence(ctx.Config.SevenTVToken, stvID, ctx.Msg.ChannelID()); err != nil {
		ctx.Reply(fmt.Sprintf("Couldn't send presence: %v", err))
		return
	}

	ctx.Reply(fmt.Sprintf("Presence updated for %s", name))
}
