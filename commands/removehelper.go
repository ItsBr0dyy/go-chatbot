package commands

import (
	"fmt"
)

func init() {
	Register(&Command{
		Name:     "removehelper",
		Category: "Bot management",
		Usage:    "removehelper <username>",
		Desc:     "Removes a user's helper access.",
		Access:   AccessOwner,
		Run:      removeHelper,
	})
}

func removeHelper(ctx *Context) {
	if !ctx.IsOwner() {
		return
	}

	if len(ctx.Args) == 0 {
		ctx.Reply(fmt.Sprintf("Usage: %sremovehelper <username>", ctx.Config.Prefix))
		return
	}

	login := cleanName(ctx.Args[0])
	if !validLogName.MatchString(login) {
		ctx.Reply("That doesn't look like a valid Twitch username.")
		return
	}

	if ctx.Config.IsOwner(login) {
		ctx.Reply(fmt.Sprintf("%s is an owner, so they can't be removed here.", login))
		return
	}

	removed, err := ctx.Helpers.Remove(login)
	if err != nil {
		ctx.Reply(fmt.Sprintf("Couldn't remove helper: %v", err))
		return
	}
	if removed {
		ctx.Reply(fmt.Sprintf("Removed %s as a helper.", login))
		return
	}

	if ctx.Config.IsHelper(login) {
		ctx.Reply(fmt.Sprintf("%s is set in configuration, so remove them there.", login))
		return
	}
	ctx.Reply(fmt.Sprintf("%s isn't a helper.", login))
}
