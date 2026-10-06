package commands

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	utils "github.com/itsbr0dyy/go-chatbot/utils/api"
)

var validLogName = regexp.MustCompile(`^[a-z0-9_]{1,25}$`)

func init() {
	Register(&Command{
		Name: "logs",
		Run:  logs,
	})
}

func cleanName(s string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(s, "@"), "#"))
}

func logs(ctx *Context) {
	channel := ctx.Msg.Channel
	user := ctx.Msg.User

	if len(ctx.Args) > 0 {
		channel = cleanName(ctx.Args[0])
	}
	if len(ctx.Args) > 1 {
		user = cleanName(ctx.Args[1])
	}

	if !validLogName.MatchString(channel) || !validLogName.MatchString(user) {
		ctx.Reply(fmt.Sprintf("Usage: %slogs [channel] [user]", ctx.Config.Prefix))
		return
	}

	month := time.Now().UTC().Format("2006-01")
	full := fmt.Sprintf("https://tv.supa.sh/logs?c=%s&u=%s&d=%s", channel, user, month)

	link, err := utils.Shorten(ctx.Config.ShortAPI, full)
	if err != nil {
		link = full
	}
	ctx.Reply(link)
}
