package commands

import (
	"errors"
	"fmt"

	utils "github.com/itsbr0dyy/go-chatbot/utils/api"
)

func init() {
	Register(&Command{Name: "mods", Run: rolesCommand("moderators", "mods", "a", "moderator")})
	Register(&Command{Name: "vips", Run: rolesCommand("vips", "vips", "a", "VIP")})
	Register(&Command{Name: "founders", Run: rolesCommand("founders", "founders", "a", "founder")})
	Register(&Command{Name: "artists", Run: rolesCommand("artists", "artists", "an", "artist")})
}

func rolesCommand(kind, cmd, article, label string) func(ctx *Context) {
	return func(ctx *Context) {
		target := ctx.Msg.User
		name := ctx.Msg.DisplayName()
		if len(ctx.Args) > 0 {
			target = cleanName(ctx.Args[0])
			name = target
			if target == ctx.Msg.User {
				name = ctx.Msg.DisplayName()
			}
		}
		if !validLogName.MatchString(target) {
			ctx.Reply(fmt.Sprintf("Usage: %s%s [username]", ctx.Config.Prefix, cmd))
			return
		}

		stats, err := utils.RolesTVStats(kind, target)
		if err != nil {
			if errors.Is(err, utils.ErrRolesNotFound) {
				ctx.Reply(fmt.Sprintf("No %s data found for %s.", label, target))
			} else {
				ctx.Reply(fmt.Sprintf("Couldn't reach roles.tv: %v", err))
			}
			return
		}

		if stats.Total == 0 {
			ctx.Reply(fmt.Sprintf("%s isn't %s %s in any channel.", name, article, label))
			return
		}

		noun := "channels"
		if stats.Total == 1 {
			noun = "channel"
		}

		link := "https://roles.tv/u/" + target
		if short, err := utils.Shorten(ctx.Config.ShortAPI, link); err == nil {
			link = short
		}

		ctx.Reply(fmt.Sprintf(
			"%s is %s %s in %d %s • %d Partners • %d Affiliates • %s total followers • %s",
			name, article, label, stats.Total, noun,
			stats.Partners, stats.Affiliates,
			utils.FormatInt(stats.Followers),
			link,
		))
	}
}
