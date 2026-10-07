package stv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type presenceData struct {
	Platform string `json:"platform"`
	ID       string `json:"id"`
}

type presenceBody struct {
	Kind    int          `json:"kind"`
	Passive bool         `json:"passive"`
	Data    presenceData `json:"data"`
}

func SendPresence(token, stvUserID, channelID string) error {
	body, err := json.Marshal(presenceBody{
		Kind:    1,
		Passive: true,
		Data:    presenceData{Platform: "TWITCH", ID: channelID},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("https://7tv.io/v3/users/%s/presences", stvUserID),
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return fmt.Errorf("7tv presence status %d: %s", resp.StatusCode, msg)
	}
	return nil
}
