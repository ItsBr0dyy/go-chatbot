package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

var (
	shortHTTP  = &http.Client{Timeout: 5 * time.Second}
	shortCache sync.Map
)

func Shorten(api, target string) (string, error) {
	if v, ok := shortCache.Load(target); ok {
		return v.(string), nil
	}

	body, err := json.Marshal(map[string]string{"url": target})
	if err != nil {
		return "", err
	}

	resp, err := shortHTTP.Post(api, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		ShortURL string `json:"shortUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.ShortURL == "" {
		return "", errors.New("redirect api returned no shortUrl")
	}

	shortCache.Store(target, data.ShortURL)
	return data.ShortURL, nil
}
