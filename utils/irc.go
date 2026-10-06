package utils

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
)

const ircAddr = "irc.chat.twitch.tv:6697"

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex
}

func Dial() (*Client, error) {
	conn, err := tls.Dial("tcp", ircAddr, nil)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, reader: bufio.NewReader(conn)}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) Send(line string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := fmt.Fprintf(c.conn, "%s\r\n", line)
	return err
}

func (c *Client) ReadLine() (string, error) {
	line, err := c.reader.ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}

func (c *Client) Login(username, oauth string) error {
	if !strings.HasPrefix(oauth, "oauth:") {
		oauth = "oauth:" + oauth
	}
	for _, l := range []string{
		"CAP REQ :twitch.tv/tags twitch.tv/commands",
		"PASS " + oauth,
		"NICK " + strings.ToLower(username),
	} {
		if err := c.Send(l); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) Join(channel string) error {
	return c.Send("JOIN #" + strings.ToLower(strings.TrimPrefix(channel, "#")))
}

func (c *Client) Part(channel string) error {
	return c.Send("PART #" + strings.ToLower(strings.TrimPrefix(channel, "#")))
}

func (c *Client) Say(channel, text string) error {
	text = strings.NewReplacer("\r", " ", "\n", " ").Replace(text)
	return c.Send("PRIVMSG #" + channel + " :" + text)
}

func (c *Client) SayReply(channel, parentID, text string) error {
	text = strings.NewReplacer("\r", " ", "\n", " ").Replace(text)
	if parentID == "" {
		return c.Send("PRIVMSG #" + channel + " :" + text)
	}
	return c.Send("@reply-parent-msg-id=" + parentID + " PRIVMSG #" + channel + " :" + text)
}

type Message struct {
	Tags    map[string]string
	User    string
	Command string
	Channel string
	Text    string
}

func (m *Message) DisplayName() string {
	if n := m.Tags["display-name"]; n != "" {
		return n
	}
	return m.User
}

func ParseMessage(line string) *Message {
	m := &Message{Tags: map[string]string{}}

	if strings.HasPrefix(line, "@") {
		i := strings.Index(line, " ")
		if i == -1 {
			return m
		}
		for _, tag := range strings.Split(line[1:i], ";") {
			if kv := strings.SplitN(tag, "=", 2); len(kv) == 2 {
				m.Tags[kv[0]] = kv[1]
			}
		}
		line = line[i+1:]
	}

	if strings.HasPrefix(line, ":") {
		i := strings.Index(line, " ")
		if i == -1 {
			return m
		}
		prefix := line[1:i]
		if j := strings.Index(prefix, "!"); j >= 0 {
			prefix = prefix[:j]
		}
		m.User = prefix
		line = line[i+1:]
	}

	head := line
	if parts := strings.SplitN(line, " :", 2); len(parts) == 2 {
		head, m.Text = parts[0], parts[1]
	}

	fields := strings.Fields(head)
	if len(fields) > 0 {
		m.Command = fields[0]
	}
	if len(fields) > 1 {
		m.Channel = strings.TrimPrefix(fields[1], "#")
	}
	return m
}
