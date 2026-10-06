package commands

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	utils "github.com/itsbr0dyy/go-chatbot/utils/api"
)

var validLastFM = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{1,14}$`)

func init() {
	Register(&Command{
		Name:     "link",
		Category: "Music",
		Usage:    "link lastfm <username>",
		Desc:     "Links your Last.fm account to your Twitch name.",
		Run:      link,
	})
}

func link(ctx *Context) {
	usage := fmt.Sprintf("Usage: %slink lastfm <username>", ctx.Config.Prefix)

	if len(ctx.Args) < 2 || strings.ToLower(ctx.Args[0]) != "lastfm" {
		ctx.Reply(usage)
		return
	}
	if ctx.Config.LastFMKey == "" {
		ctx.Reply("Last.fm isn't set up on this bot.")
		return
	}

	name := ctx.Args[1]
	if !validLastFM.MatchString(name) {
		ctx.Reply("That doesn't look like a valid Last.fm username.")
		return
	}

	if _, err := utils.LastFMRecent(ctx.Config.LastFMKey, name); err != nil &&
		!errors.Is(err, utils.ErrNoTracks) {
		if errors.Is(err, utils.ErrLastFMUserNotFound) {
			ctx.Reply(fmt.Sprintf("Last.fm user %q not found.", name))
		} else {
			ctx.Reply(fmt.Sprintf("Couldn't check that account: %v", err))
		}
		return
	}

	if err := ctx.Links.Set(ctx.Msg.User, name); err != nil {
		ctx.Reply(fmt.Sprintf("Couldn't save your link: %v", err))
		return
	}
	ctx.Reply(fmt.Sprintf("Linked your Last.fm account: %s", name))
}
