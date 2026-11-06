package commands

import (
	"fmt"
	"strings"

	"github.com/itsbr0dyy/go-chatbot/utils"
)

type Status int

const (
	StatusDefault Status = iota
	StatusMod
	StatusHelper
	StatusOwner
)

func (s Status) String() string {
	switch s {
	case StatusMod:
		return "mod"
	case StatusHelper:
		return "helper"
	case StatusOwner:
		return "owner"
	default:
		return "default"
	}
}

func (c *Context) statusOf(login string) Status {
	switch {
	case c.Config.IsOwner(login):
		return StatusOwner
	case c.isHelperLogin(login):
		return StatusHelper
	}
	return StatusDefault
}

func (c *Context) statusIn(channel, login string) Status {
	s := c.statusOf(login)
	if s < StatusMod && c.Mods.Has(channel, login) {
		return StatusMod
	}
	return s
}

func (c *Context) MyStatus() Status {
	s := c.statusIn(c.Msg.Channel, c.Msg.User)
	if s < StatusMod && isChannelMod(c.Msg) {
		return StatusMod
	}
	return s
}

func isChannelMod(m *utils.Message) bool {
	if m.Tags["mod"] == "1" || strings.EqualFold(m.User, m.Channel) {
		return true
	}
	for _, b := range strings.Split(m.Tags["badges"], ",") {
		if strings.HasPrefix(b, "moderator/") || strings.HasPrefix(b, "broadcaster/") {
			return true
		}
	}
	return false
}

func isStatusWord(s string) bool {
	switch s {
	case "helper", "mod", "default", "owner":
		return true
	}
	return false
}

func init() {
	Register(&Command{
		Name:     "status",
		Category: "Bot management",
		Usage:    "status [user] [helper|mod|default]",
		Desc:     "Shows your status or another user's. Owners can set a user's status to helper (everywhere), mod (this channel only), or default (reset).",
		Run:      statusCmd,
	})
}

func statusCmd(ctx *Context) {
	usage := fmt.Sprintf("Usage: %sstatus [user] [helper|mod|default]", ctx.Config.Prefix)
	channel := ctx.Msg.Channel

	// >status -> your status here
	if len(ctx.Args) == 0 {
		s := ctx.MyStatus()
		note := ""
		if s == StatusMod {
			note = " (in this channel)"
		}
		ctx.Reply(fmt.Sprintf("Your status: %s%s", s, note))
		return
	}

	if len(ctx.Args) == 1 {
		target := cleanName(ctx.Args[0])
		if !validLogName.MatchString(target) {
			ctx.Reply(usage)
			return
		}
		s := ctx.statusIn(channel, target)
		note := ""
		if s == StatusMod {
			note = " (in this channel)"
		}
		ctx.Reply(fmt.Sprintf("%s's status: %s%s", target, s, note))
		return
	}

	a, b := strings.ToLower(ctx.Args[0]), strings.ToLower(ctx.Args[1])
	if isStatusWord(a) && !isStatusWord(b) {
		a, b = b, a
	}
	target, word := cleanName(a), b

	if !ctx.IsOwner() {
		ctx.Reply("Only owners can change statuses.")
		return
	}
	if !validLogName.MatchString(target) || !isStatusWord(word) {
		ctx.Reply(usage)
		return
	}
	if word == "owner" {
		ctx.Reply("Owners are set in config.")
		return
	}
	if ctx.Config.IsOwner(target) {
		ctx.Reply(fmt.Sprintf("%s is an owner, set in.json.", target))
		return
	}
	if ctx.Config.IsHelper(target) {
		if word == "helper" {
			ctx.Reply(fmt.Sprintf("%s is already a helper.", target))
		} else {
			ctx.Reply(fmt.Sprintf("%s is a helper set in config, change them there.", target))
		}
		return
	}

	fail := func(err error) {
		ctx.Reply(fmt.Sprintf("Couldn't update that status: %v", err))
	}

	switch word {
	case "helper":
		if _, err := ctx.Mods.Remove(channel, target); err != nil {
			fail(err)
			return
		}
		added, err := ctx.Helpers.Add(target, ctx.Msg.User)
		if err != nil {
			fail(err)
			return
		}
		if !added {
			ctx.Reply(fmt.Sprintf("%s is already a helper.", target))
			return
		}
		ctx.Reply(fmt.Sprintf("Set %s's status to helper.", target))

	case "mod":
		if _, err := ctx.Helpers.Remove(target); err != nil {
			fail(err)
			return
		}
		added, err := ctx.Mods.Add(channel, target, ctx.Msg.User)
		if err != nil {
			fail(err)
			return
		}
		if !added {
			ctx.Reply(fmt.Sprintf("%s is already a mod in #%s.", target, channel))
			return
		}
		ctx.Reply(fmt.Sprintf("Set %s's status to mod in #%s.", target, channel))

	case "default":
		wasHelper, err := ctx.Helpers.Remove(target)
		if err != nil {
			fail(err)
			return
		}
		wasMod, err := ctx.Mods.Remove(channel, target)
		if err != nil {
			fail(err)
			return
		}
		if !wasHelper && !wasMod {
			ctx.Reply(fmt.Sprintf("%s already has the default status here.", target))
			return
		}
		ctx.Reply(fmt.Sprintf("Reset %s's status to default.", target))
	}
}
