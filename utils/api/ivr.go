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

type TwitchUser struct {
	Banned       bool      `json:"banned"`
	DisplayName  string    `json:"displayName"`
	Login        string    `json:"login"`
	ID           string    `json:"id"`
	Bio          string    `json:"bio"`
	Follows      *int      `json:"follows"`
	Followers    int       `json:"followers"`
	ChatterCount *int      `json:"chatterCount"`
	CreatedAt    time.Time `json:"createdAt"`
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

func FormatAgo(since time.Time) string {
	d := time.Since(since)
	if d < 0 {
		d = 0
	}

	days := int(d.Hours() / 24)
	units := []struct {
		n    int
		name string
	}{
		{days / 365, "y"},
		{(days % 365) / 30, "mo"},
		{(days % 365) % 30, "d"},
		{int(d.Hours()) % 24, "h"},
		{int(d.Minutes()) % 60, "m"},
	}

	for i, u := range units {
		if u.n == 0 {
			continue
		}
		out := fmt.Sprintf("%d%s", u.n, u.name)
		if i+1 < len(units) && units[i+1].n > 0 {
			out += fmt.Sprintf(" %d%s", units[i+1].n, units[i+1].name)
		}
		return out
	}
	return "<1m"
}
