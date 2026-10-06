package commands

import (
	"fmt"
)

func init() {
	Register(&Command{
		Name: "addhelper",
		Run:  addHelper,
	})
}

func addHelper(ctx *Context) {
	if !ctx.IsOwner() {
		return
	}

	if len(ctx.Args) == 0 {
		ctx.Reply(fmt.Sprintf("Usage: %saddhelper <username>", ctx.Config.Prefix))
		return
	}

	login := cleanName(ctx.Args[0])
	if !validLogName.MatchString(login) {
		ctx.Reply("That doesn't look like a valid Twitch username.")
		return
	}

	if ctx.Config.IsOwner(login) || ctx.isHelperLogin(login) {
		ctx.Reply(fmt.Sprintf("%s is already a helper.", login))
		return
	}

	if _, err := ctx.Helpers.Add(login, ctx.Msg.User); err != nil {
		ctx.Reply(fmt.Sprintf("Couldn't save helper: %v", err))
		return
	}
	ctx.Reply(fmt.Sprintf("Added %s as a helper.", login))
}
