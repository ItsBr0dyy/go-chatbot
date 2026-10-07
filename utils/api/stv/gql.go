package stv

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var ErrUserNotFound = errors.New("7tv user not found")

var httpClient = &http.Client{Timeout: 5 * time.Second}

const gqlURL = "https://7tv.io/v4/gql"

const userByConnectionQuery = `
query GetUserByConnection($platformId: String!) {
  users {
    userByConnection(platform: TWITCH, platformId: $platformId) {
      id
    }
  }
}`

func UserIDFromTwitch(twitchID string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"query":     userByConnectionQuery,
		"variables": map[string]string{"platformId": twitchID},
	})
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Post(gqlURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Data struct {
			Users struct {
				UserByConnection *struct {
					ID string `json:"id"`
				} `json:"userByConnection"`
			} `json:"users"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if len(data.Errors) > 0 {
		return "", fmt.Errorf("7tv gql: %s", data.Errors[0].Message)
	}
	u := data.Data.Users.UserByConnection
	if u == nil || u.ID == "" {
		return "", ErrUserNotFound
	}
	return u.ID, nil
}
