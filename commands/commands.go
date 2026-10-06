package commands

import (
	"log"
	"strings"

	"github.com/itsbr0dyy/go-chatbot/config"
	"github.com/itsbr0dyy/go-chatbot/utils"
)

type Access int

const (
	AccessEveryone Access = iota
	AccessHelper
	AccessOwner
)

func (a Access) String() string {
	switch a {
	case AccessHelper:
		return "Helper"
	case AccessOwner:
		return "Owner"
	default:
		return "Everyone"
	}
}

type Context struct {
	Client   *utils.Client
	Msg      *utils.Message
	Args     []string
	Config   *config.Config
	Channels *config.Channels
	Links    *config.Links
	Helpers  *config.Helpers
}

func (c *Context) Reply(text string) {
	if err := c.Client.SayReply(c.Msg.Channel, c.Msg.Tags["id"], text); err != nil {
		log.Printf("reply failed: %v", err)
		return
	}
	log.Printf("-> #%s: %s", c.Msg.Channel, text)
}

func (c *Context) IsOwner() bool {
	return c.Config.IsOwner(c.Msg.User)
}

func (c *Context) IsHelper() bool {
	return c.IsOwner() || c.isHelperLogin(c.Msg.User)
}

func (c *Context) isHelperLogin(login string) bool {
	return c.Config.IsHelper(login) || c.Helpers.Has(login)
}

type Command struct {
	Name     string
	Aliases  []string
	Category string
	Usage    string
	Desc     string
	Access   Access
	Run      func(ctx *Context)
}

var registry = map[string]*Command{}

func Register(cmd *Command) {
	registry[strings.ToLower(cmd.Name)] = cmd
	for _, a := range cmd.Aliases {
		registry[strings.ToLower(a)] = cmd
	}
}

func Handle(client *utils.Client, msg *utils.Message, cfg *config.Config, channels *config.Channels, links *config.Links, helpers *config.Helpers) {
	if !strings.HasPrefix(msg.Text, cfg.Prefix) {
		return
	}

	fields := strings.Fields(strings.TrimPrefix(msg.Text, cfg.Prefix))
	if len(fields) == 0 {
		return
	}

	cmd, ok := registry[strings.ToLower(fields[0])]
	if !ok {
		return
	}

	ctx := &Context{
		Client:   client,
		Msg:      msg,
		Args:     fields[1:],
		Config:   cfg,
		Channels: channels,
		Links:    links,
		Helpers:  helpers,
	}

	switch cmd.Access {
	case AccessOwner:
		if !ctx.IsOwner() {
			return
		}
	case AccessHelper:
		if !ctx.IsHelper() {
			return
		}
	}

	log.Printf("command %q from %s in #%s", fields[0], msg.User, msg.Channel)
	utils.CountCommand()
	cmd.Run(ctx)
}
