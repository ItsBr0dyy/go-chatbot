package commands

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	utils "github.com/itsbr0dyy/go-chatbot/utils/api"
)

var (
	validLastFM = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{1,14}$`)
	validMCName = regexp.MustCompile(`^[A-Za-z0-9_]{3,16}$`)
)

func init() {
	Register(&Command{
		Name:     "link",
		Category: "Accounts",
		Usage:    "link <lastfm|mc> <username>",
		Desc:     "Links your Last.fm or Minecraft account to your Twitch name.",
		Run:      link,
	})
}

func link(ctx *Context) {
	usage := fmt.Sprintf("Usage: %slink <lastfm|mc> <username>", ctx.Config.Prefix)

	if len(ctx.Args) < 2 {
		ctx.Reply(usage)
		return
	}

	switch strings.ToLower(ctx.Args[0]) {
	case "lastfm":
		linkLastFM(ctx, ctx.Args[1])
	case "mc", "minecraft":
		linkMC(ctx, ctx.Args[1])
	default:
		ctx.Reply(usage)
	}
}

func linkLastFM(ctx *Context, name string) {
	if ctx.Config.LastFMKey == "" {
		ctx.Reply("Last.fm isn't set up on this bot.")
		return
	}
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

	if err := ctx.Links.Set("lastfm", ctx.Msg.User, name); err != nil {
		ctx.Reply(fmt.Sprintf("Couldn't save your link: %v", err))
		return
	}
	ctx.Reply(fmt.Sprintf("Linked your Last.fm account: %s", name))
}

func linkMC(ctx *Context, name string) {
	name = strings.TrimPrefix(name, "@")
	if !validMCName.MatchString(name) {
		ctx.Reply("That doesn't look like a valid Minecraft username.")
		return
	}

	u, err := utils.MCSRLookup(name)
	if err != nil {
		if errors.Is(err, utils.ErrMCSRNotFound) {
			ctx.Reply(fmt.Sprintf("No MCSR Ranked profile found for %s.", name))
		} else {
			ctx.Reply(fmt.Sprintf("Couldn't check that account: %v", err))
		}
		return
	}

	if err := ctx.Links.Set("mc", ctx.Msg.User, u.Nickname); err != nil {
		ctx.Reply(fmt.Sprintf("Couldn't save your link: %v", err))
		return
	}
	ctx.Reply(fmt.Sprintf("Linked your Minecraft account: %s", u.Nickname))
}
