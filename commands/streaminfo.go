package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/itsbr0dyy/go-chatbot/utils"
)

func init() {
	Register(&Command{
		Name:     "streaminfo",
		Aliases:  []string{"si"},
		Category: "Twitch info",
		Usage:    "streaminfo [channel]",
		Desc:     "Shows whether a channel is live, with its title, game, viewers and uptime. Uses this channel if you leave the name out.",
		Run:      streamInfoCmd,
	})
	Register(&Command{
		Name:     "title",
		Category: "Twitch info",
		Usage:    "title [channel]",
		Desc:     "Shows a channel's current stream title, or its last one if it's offline.",
		Run:      titleCmd,
	})
}

func streamTarget(ctx *Context, cmd string) (*utils.TwitchUser, bool) {
	channel := ctx.Msg.Channel
	if len(ctx.Args) > 0 {
		channel = cleanName(ctx.Args[0])
	}
	if !validLogName.MatchString(channel) {
		ctx.Reply(fmt.Sprintf("Usage: %s%s [channel]", ctx.Config.Prefix, cmd))
		return nil, false
	}

	u, err := utils.IVRUser(channel)
	if err != nil {
		if errors.Is(err, utils.ErrTwitchUserNotFound) {
			ctx.Reply(fmt.Sprintf("Couldn't find a Twitch channel named %q.", channel))
		} else {
			ctx.Reply(fmt.Sprintf("Couldn't look that channel up: %v", err))
		}
		return nil, false
	}
	return u, true
}

func displayOr(u *utils.TwitchUser) string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Login
}

func streamInfoCmd(ctx *Context) {
	u, ok := streamTarget(ctx, "streaminfo")
	if !ok {
		return
	}
	name := displayOr(u)

	if s := u.LiveStream(); s != nil {
		parts := []string{name + " is live"}
		if s.Title != "" {
			parts = append(parts, `"`+s.Title+`"`)
		}
		if s.Game != "" {
			parts = append(parts, s.Game)
		}
		if s.Viewers > 0 {
			noun := "viewers"
			if s.Viewers == 1 {
				noun = "viewer"
			}
			parts = append(parts, fmt.Sprintf("%s %s", utils.FormatInt(s.Viewers), noun))
		}
		if !s.StartedAt.IsZero() {
			parts = append(parts, "live for "+utils.FormatAgo(s.StartedAt))
		}
		ctx.Reply(strings.Join(parts, " • "))
		return
	}

	parts := []string{name + " is offline"}
	if lb := u.LastBroadcast; lb != nil {
		if !lb.StartedAt.IsZero() {
			parts = append(parts, "last live "+utils.FormatAgo(lb.StartedAt)+" ago")
		}
		if lb.Title != "" {
			parts = append(parts, `last title: "`+lb.Title+`"`)
		}
	}
	ctx.Reply(strings.Join(parts, " • "))
}

func titleCmd(ctx *Context) {
	u, ok := streamTarget(ctx, "title")
	if !ok {
		return
	}
	name := displayOr(u)

	if s := u.LiveStream(); s != nil && s.Title != "" {
		ctx.Reply(fmt.Sprintf(`%s's title: "%s"`, name, s.Title))
		return
	}
	if lb := u.LastBroadcast; lb != nil && lb.Title != "" {
		if u.IsLive() {
			ctx.Reply(fmt.Sprintf(`%s is live, last known title: "%s"`, name, lb.Title))
		} else {
			ctx.Reply(fmt.Sprintf(`%s is offline, last title: "%s"`, name, lb.Title))
		}
		return
	}
	ctx.Reply(fmt.Sprintf("%s doesn't have a stream title.", name))
}
