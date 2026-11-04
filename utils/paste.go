package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var pasteHTTP = &http.Client{Timeout: 8 * time.Second}

func Paste(api, text string) (string, error) {
	resp, err := pasteHTTP.Post(api, "text/plain; charset=utf-8", strings.NewReader(text))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("paste api returned %d", resp.StatusCode)
	}

	s := strings.TrimSpace(string(body))
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s, nil
	}

	var data struct {
		URL      string `json:"url"`
		ShortURL string `json:"shortUrl"`
		Link     string `json:"link"`
	}
	if json.Unmarshal(body, &data) == nil {
		for _, u := range []string{data.URL, data.ShortURL, data.Link} {
			if u != "" {
				return u, nil
			}
		}
	}

	if len(s) > 80 {
		s = s[:80]
	}
	return "", fmt.Errorf("unexpected paste response: %q", s)
}
