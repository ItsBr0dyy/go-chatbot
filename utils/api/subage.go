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

var ErrSubAgeNotFound = errors.New("user or channel not found")

type SubAge struct {
	User struct {
		DisplayName string `json:"displayName"`
		Login       string `json:"login"`
	} `json:"user"`
	Channel struct {
		DisplayName string `json:"displayName"`
		Login       string `json:"login"`
	} `json:"channel"`
	StatusHidden bool `json:"statusHidden"`
	Hidden       bool `json:"hidden"`
	Streak       *struct {
		Months int `json:"months"`
	} `json:"streak"`
	Cumulative *struct {
		Months int `json:"months"`
	} `json:"cumulative"`
	Meta *struct {
		Type     string `json:"type"`
		Tier     string `json:"tier"`
		EndsAt   string `json:"endsAt"`
		RenewsAt string `json:"renewsAt"`
		Gift     *struct {
			IsGift      bool   `json:"isGift"`
			Name        string `json:"name"`
			Login       string `json:"login"`
			DisplayName string `json:"displayName"`
		} `json:"gift"`
	} `json:"meta"`
}

var subAgeHTTP = &http.Client{Timeout: 5 * time.Second}

func IVRSubAge(user, channel string) (*SubAge, error) {
	u := fmt.Sprintf("https://api.ivr.fi/v2/twitch/subage/%s/%s", url.PathEscape(user), url.PathEscape(channel))
	resp, err := subAgeHTTP.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrSubAgeNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ivr returned %d", resp.StatusCode)
	}

	var s SubAge
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func parseTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s)
	return t, err == nil
}

func (s *SubAge) IsHidden() bool { return s.Hidden || s.StatusHidden }

func (s *SubAge) Subscribed() bool {
	if s.Meta == nil {
		return false
	}
	if end, ok := parseTime(s.Meta.EndsAt); ok && end.Before(time.Now()) {
		return false
	}
	return true
}

func (s *SubAge) Months() int {
	if s.Cumulative != nil && s.Cumulative.Months > 0 {
		return s.Cumulative.Months
	}
	if s.Streak != nil {
		return s.Streak.Months
	}
	return 0
}

func (s *SubAge) SubType() string {
	if s.Meta == nil {
		return ""
	}

	tier := ""
	switch strings.TrimSpace(s.Meta.Tier) {
	case "1000", "1":
		tier = "Tier 1"
	case "2000", "2":
		tier = "Tier 2"
	case "3000", "3":
		tier = "Tier 3"
	}

	if strings.EqualFold(s.Meta.Type, "prime") {
		return "Prime"
	}
	if tier == "" {
		return "sub"
	}
	return tier
}

func (s *SubAge) Gifter() (string, bool) {
	if s.Meta == nil {
		return "", false
	}
	isGift := strings.EqualFold(s.Meta.Type, "gift")
	name := ""
	if g := s.Meta.Gift; g != nil {
		isGift = isGift || g.IsGift
		name = g.DisplayName
		if name == "" {
			name = g.Name
		}
		if name == "" {
			name = g.Login
		}
	}
	return name, isGift
}

func (s *SubAge) Remaining() (time.Duration, bool, bool) {
	if s.Meta == nil {
		return 0, false, false
	}
	if t, ok := parseTime(s.Meta.RenewsAt); ok {
		return time.Until(t), true, true
	}
	if t, ok := parseTime(s.Meta.EndsAt); ok {
		return time.Until(t), false, true
	}
	return 0, false, false
}

func FormatLong(d time.Duration) string {
	if d < time.Minute {
		return "less than a minute"
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	var parts []string
	add := func(n int, unit string) {
		if n == 0 {
			return
		}
		if n != 1 {
			unit += "s"
		}
		parts = append(parts, fmt.Sprintf("%d %s", n, unit))
	}
	add(days, "day")
	add(hours, "hour")
	add(mins, "minute")

	switch len(parts) {
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
	}
}
