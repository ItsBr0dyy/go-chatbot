package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type EventBadge struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	StartAt string `json:"startAt"`
	EndAt   string `json:"endAt"`
	Free    bool   `json:"free"`
}

func parseBadgeTime(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (b EventBadge) Start() (time.Time, bool) { return parseBadgeTime(b.StartAt) }
func (b EventBadge) End() (time.Time, bool)   { return parseBadgeTime(b.EndAt) }

var catHTTP = &http.Client{Timeout: 8 * time.Second}

func catGet(u string) ([]EventBadge, error) {
	resp, err := catHTTP.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catquery returned %d", resp.StatusCode)
	}

	var data struct {
		Badges []EventBadge `json:"badges"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data.Badges, nil
}

func CatQueryBadges(user string) (all, missing []EventBadge, err error) {
	var allErr, missErr error

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		all, allErr = catGet("https://api.catquery.com/eventBadges")
	}()
	go func() {
		defer wg.Done()
		missing, missErr = catGet("https://api.catquery.com/user/missingEventBadges?name=" + url.QueryEscape(user))
	}()
	wg.Wait()

	if allErr != nil {
		return nil, nil, allErr
	}
	if missErr != nil {
		return nil, nil, missErr
	}
	if len(all) == 0 && len(missing) == 0 {
		return nil, nil, errors.New("empty response")
	}
	return all, missing, nil
}
