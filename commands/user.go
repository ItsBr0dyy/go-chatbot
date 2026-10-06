package commands

import (
	"errors"
	"fmt"
	"strings"

	utils "github.com/itsbr0dyy/go-chatbot/utils/api"
)

func init() {
	Register(&Command{
		Name:     "user",
		Aliases:  []string{"u"},
		Category: "Twitch info",
		Usage:    "user [username]",
		Desc:     "Shows Twitch account info.",
		Run:      user,
	})
}

func user(ctx *Context) {
	target := ctx.Msg.User
	if len(ctx.Args) > 0 {
		target = cleanName(ctx.Args[0])
	}
	if !validLogName.MatchString(target) {
		ctx.Reply(fmt.Sprintf("Usage: %suser [username]", ctx.Config.Prefix))
		return
	}

	u, err := utils.IVRUser(target)
	if err != nil {
		if errors.Is(err, utils.ErrTwitchUserNotFound) {
			ctx.Reply(fmt.Sprintf("Couldn't find a Twitch user named %q.", target))
		} else {
			ctx.Reply(fmt.Sprintf("Couldn't look that user up: %v", err))
		}
		return
	}

	name := u.DisplayName
	if u.Banned {
		name += " (banned)"
	}
	parts := []string{name}

	if roles := u.RoleNames(); len(roles) > 0 {
		parts = append(parts, strings.Join(roles, ", "))
	}

	parts = append(parts, fmt.Sprintf("%d followers", u.Followers))
	if u.Follows != nil {
		parts = append(parts, fmt.Sprintf("following %d", *u.Follows))
	}

	parts = append(parts, "ID "+u.ID)

	if !u.CreatedAt.IsZero() {
		parts = append(parts, "created "+utils.FormatAgo(u.CreatedAt)+" ago")
	}

	if u.ChatterCount != nil {
		parts = append(parts, fmt.Sprintf("%d chatters", *u.ChatterCount))
	}

	if u.LastBroadcast != nil && !u.LastBroadcast.StartedAt.IsZero() {
		ago := utils.FormatAgo(u.LastBroadcast.StartedAt)
		if u.IsLive() {
			parts = append(parts, "live now, started "+ago+" ago")
		} else {
			parts = append(parts, "last live "+ago+" ago")
		}
	} else if u.IsLive() {
		parts = append(parts, "live now")
	}

	if bio := shortBio(u.Bio, 60); bio != "" {
		parts = append(parts, fmt.Sprintf("bio: %q", bio))
	}

	ctx.Reply(strings.Join(parts, " • "))
}

func shortBio(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}
