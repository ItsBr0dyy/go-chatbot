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

type Deps struct {
	Config    *config.Config
	Channels  *config.Channels
	Links     *config.Links
	Helpers   *config.Helpers
	Reminders *config.Reminders
	Cats      *config.CatStore
}

type Context struct {
	Client    *utils.Client
	Msg       *utils.Message
	Args      []string
	Rest      string
	Config    *config.Config
	Channels  *config.Channels
	Links     *config.Links
	Helpers   *config.Helpers
	Reminders *config.Reminders
	Cats      *config.CatStore
}

func (c *Context) Reply(text string) {
	if err := c.Client.SayReply(c.Msg.Channel, c.Msg.Tags["id"], text); err != nil {
		log.Printf("reply failed: %v", err)
	}
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

func Handle(client *utils.Client, msg *utils.Message, d *Deps) {
	body, ok := commandText(msg.Text, d.Config)
	if !ok {
		return
	}

	fields := strings.Fields(body)
	if len(fields) == 0 {
		return
	}

	cmd, ok := registry[strings.ToLower(fields[0])]
	if !ok {
		return
	}

	rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(body), fields[0]))

	ctx := &Context{
		Client:    client,
		Msg:       msg,
		Args:      fields[1:],
		Rest:      rest,
		Config:    d.Config,
		Channels:  d.Channels,
		Links:     d.Links,
		Helpers:   d.Helpers,
		Reminders: d.Reminders,
		Cats:      d.Cats,
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

	utils.CountCommand()
	cmd.Run(ctx)
}

func commandText(text string, cfg *config.Config) (string, bool) {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\U000e0000", ""))

	if strings.HasPrefix(text, cfg.Prefix) {
		return strings.TrimPrefix(text, cfg.Prefix), true
	}

	mention := "@" + cfg.Username
	if len(text) < len(mention) || !strings.EqualFold(text[:len(mention)], mention) {
		return "", false
	}

	rest := strings.TrimLeft(text[len(mention):], " ,:")
	rest = strings.TrimPrefix(rest, cfg.Prefix)
	return rest, true
}
