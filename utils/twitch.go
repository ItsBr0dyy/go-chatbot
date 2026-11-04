package utils

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

func TwitchIDByLogin(login string) (string, error) {
	resp, err := lastfmHTTP.Get("https://api.ivr.fi/v2/twitch/user?login=" + url.QueryEscape(login))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("user not found")
	}

	var users []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return "", err
	}
	if len(users) == 0 || users[0].ID == "" {
		return "", errors.New("user not found")
	}
	return users[0].ID, nil
}
