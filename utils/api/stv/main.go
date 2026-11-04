package stv

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrTokenInvalid = errors.New("7tv token invalid or expired")
	ErrUserNotFound = errors.New("no 7tv account for that twitch user")
)

var httpClient = &http.Client{Timeout: 8 * time.Second}

type Client struct {
	token string
}

func New(token string) *Client {
	return &Client{token: strings.TrimSpace(token)}
}

type ChannelEmote struct {
	ID          string
	Name        string
	DefaultName string
}

func (c *Client) ChannelEmotes(stvUserID string) ([]ChannelEmote, error) {
	resp, err := httpClient.Get("https://7tv.io/v3/users/" + url.PathEscape(stvUserID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("7tv returned %d", resp.StatusCode)
	}

	var data struct {
		Connections []struct {
			Platform string `json:"platform"`
			EmoteSet *struct {
				Emotes []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Data struct {
						Name string `json:"name"`
					} `json:"data"`
				} `json:"emotes"`
			} `json:"emote_set"`
		} `json:"connections"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	for _, conn := range data.Connections {
		if conn.Platform != "TWITCH" || conn.EmoteSet == nil {
			continue
		}
		out := make([]ChannelEmote, 0, len(conn.EmoteSet.Emotes))
		for _, e := range conn.EmoteSet.Emotes {
			def := e.Data.Name
			if def == "" {
				def = e.Name
			}
			out = append(out, ChannelEmote{ID: e.ID, Name: e.Name, DefaultName: def})
		}
		return out, nil
	}
	return nil, nil
}

func UserIDFromTwitch(twitchID string) (string, error) {
	return New("").UserIDFromTwitch(twitchID)
}
