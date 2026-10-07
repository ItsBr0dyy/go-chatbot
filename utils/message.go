package utils

import (
	"strconv"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v4"
)

type Message struct {
	Tags    map[string]string
	User    string
	Channel string
	Text    string
}

func NewMessage(m twitch.PrivateMessage) *Message {
	return &Message{
		Tags:    m.Tags,
		User:    m.User.Name,
		Channel: m.Channel,
		Text:    m.Message,
	}
}

func (m *Message) DisplayName() string {
	if n := m.Tags["display-name"]; n != "" {
		return n
	}
	return m.User
}

func (m *Message) ID() string        { return m.Tags["id"] }
func (m *Message) UserID() string    { return m.Tags["user-id"] }
func (m *Message) ChannelID() string { return m.Tags["room-id"] }

func (m *Message) HasBadge(name string) bool {
	for _, b := range strings.Split(m.Tags["badges"], ",") {
		if strings.SplitN(b, "/", 2)[0] == name {
			return true
		}
	}
	return false
}

func (m *Message) IsBroadcaster() bool {
	return m.HasBadge("broadcaster") || m.User == m.Channel
}

func (m *Message) IsMod() bool {
	return m.Tags["mod"] == "1" || m.HasBadge("moderator") || m.IsBroadcaster()
}

func (m *Message) IsVIP() bool {
	return m.HasBadge("vip")
}

func (m *Message) IsSubscriber() bool {
	return m.Tags["subscriber"] == "1" || m.HasBadge("subscriber") || m.HasBadge("founder")
}

func (m *Message) SentAt() (time.Time, bool) {
	ms, err := strconv.ParseInt(m.Tags["tmi-sent-ts"], 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.UnixMilli(ms), true
}
