package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrTwitchUserNotFound = errors.New("twitch user not found")

type TwitchBadge struct {
	SetID       string `json:"setID"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type TwitchUser struct {
	Banned       bool          `json:"banned"`
	DisplayName  string        `json:"displayName"`
	Login        string        `json:"login"`
	ID           string        `json:"id"`
	Bio          string        `json:"bio"`
	Follows      *int          `json:"follows"`
	Followers    int           `json:"followers"`
	Badges       []TwitchBadge `json:"badges"`
	ChatterCount *int          `json:"chatterCount"`
	CreatedAt    time.Time     `json:"createdAt"`
	Roles        struct {
		IsPreAffiliate *bool `json:"isPreAffiliate"`
		IsAffiliate    *bool `json:"isAffiliate"`
		IsPartner      *bool `json:"isPartner"`
		IsStaff        *bool `json:"isStaff"`
	} `json:"roles"`
	Stream        json.RawMessage `json:"stream"`
	LastBroadcast *struct {
		StartedAt time.Time `json:"startedAt"`
		Title     string    `json:"title"`
	} `json:"lastBroadcast"`
}

func (u *TwitchUser) IsLive() bool {
	s := strings.TrimSpace(string(u.Stream))
	return s != "" && s != "null"
}

func (u *TwitchUser) RoleNames() []string {
	var roles []string
	if u.Roles.IsStaff != nil && *u.Roles.IsStaff {
		roles = append(roles, "Staff")
	}
	if u.Roles.IsPartner != nil && *u.Roles.IsPartner {
		roles = append(roles, "Partner")
	}
	if u.Roles.IsAffiliate != nil && *u.Roles.IsAffiliate {
		roles = append(roles, "Affiliate")
	}
	if u.Roles.IsPreAffiliate != nil && *u.Roles.IsPreAffiliate {
		roles = append(roles, "Pre-affiliate")
	}
	return roles
}

var ivrHTTP = &http.Client{Timeout: 5 * time.Second}

func IVRUser(login string) (*TwitchUser, error) {
	resp, err := ivrHTTP.Get("https://api.ivr.fi/v2/twitch/user?login=" + url.QueryEscape(login))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrTwitchUserNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ivr returned %d", resp.StatusCode)
	}

	var users []TwitchUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, ErrTwitchUserNotFound
	}
	return &users[0], nil
}

type StreamInfo struct {
	Title     string
	Game      string
	Viewers   int
	StartedAt time.Time
}

func (u *TwitchUser) LiveStream() *StreamInfo {
	if !u.IsLive() {
		return nil
	}

	var raw struct {
		Title        string          `json:"title"`
		ViewersCount int             `json:"viewersCount"`
		Viewers      int             `json:"viewers"`
		CreatedAt    string          `json:"createdAt"`
		StartedAt    string          `json:"startedAt"`
		Game         json.RawMessage `json:"game"`
	}
	info := &StreamInfo{}
	if err := json.Unmarshal(u.Stream, &raw); err != nil {
		return info
	}

	info.Title = raw.Title
	info.Viewers = raw.ViewersCount
	if info.Viewers == 0 {
		info.Viewers = raw.Viewers
	}

	for _, s := range []string{raw.CreatedAt, raw.StartedAt} {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			info.StartedAt = t
			break
		}
	}

	if len(raw.Game) > 0 {
		var obj struct {
			DisplayName string `json:"displayName"`
			Name        string `json:"name"`
		}
		var s string
		switch {
		case json.Unmarshal(raw.Game, &obj) == nil && (obj.DisplayName != "" || obj.Name != ""):
			info.Game = obj.DisplayName
			if info.Game == "" {
				info.Game = obj.Name
			}
		case json.Unmarshal(raw.Game, &s) == nil:
			info.Game = s
		}
	}
	return info
}
