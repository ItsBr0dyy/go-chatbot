package utils

import (
	"strings"

	"github.com/gempir/go-twitch-irc/v4"
)

type Client struct {
	irc *twitch.Client
}

func NewClient(username, oauth string) *Client {
	if !strings.HasPrefix(oauth, "oauth:") {
		oauth = "oauth:" + oauth
	}
	return &Client{irc: twitch.NewClient(strings.ToLower(username), oauth)}
}

func (c *Client) IRC() *twitch.Client { return c.irc }

func (c *Client) Connect() error { return c.irc.Connect() }

func normalize(channel string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(channel), "#"))
}

func (c *Client) Join(channel string) error {
	c.irc.Join(normalize(channel))
	return nil
}

func (c *Client) Part(channel string) error {
	c.irc.Depart(normalize(channel))
	return nil
}

func (c *Client) SayReply(channel, parentID, text string) error {
	text = strings.NewReplacer("\r", " ", "\n", " ").Replace(text)
	if parentID == "" {
		c.irc.Say(channel, text)
	} else {
		c.irc.Reply(channel, parentID, text)
	}
	return nil
}

type Message struct {
	Tags    map[string]string
	User    string
	Channel string
	Text    string
}

func (m *Message) DisplayName() string {
	if n := m.Tags["display-name"]; n != "" {
		return n
	}
	return m.User
}
