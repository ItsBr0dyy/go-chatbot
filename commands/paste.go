package commands

import (
	"fmt"

	"github.com/itsbr0dyy/go-chatbot/utils"
)

const (
	pasteSite = "https://itsbr0dyy.dev/h"
	pasteAPI  = "https://itsbr0dyy.dev/api/haste"
)

func init() {
	Register(&Command{
		Name:     "paste",
		Aliases:  []string{"haste"},
		Category: "Utility",
		Usage:    "paste [message]",
		Desc:     "Uploads your message to itsbr0dyy.dev/h and links it. With no message, links the paste site.",
		Run:      pasteCmd,
	})
}

func pasteCmd(ctx *Context) {
	if ctx.Rest == "" {
		ctx.Reply(pasteSite)
		return
	}

	link, err := utils.Paste(pasteAPI, ctx.Rest)
	if err != nil {
		ctx.Reply(fmt.Sprintf("Couldn't create the paste: %v", err))
		return
	}
	ctx.Reply(link)
}
