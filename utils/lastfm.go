package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var (
	ErrLastFMUserNotFound = errors.New("last.fm user not found")
	ErrNoTracks           = errors.New("no scrobbles found")
)

type Track struct {
	Name       string
	Artist     string
	Album      string
	URL        string
	NowPlaying bool
}

var lastfmHTTP = &http.Client{Timeout: 5 * time.Second}

func lastfmGet(q url.Values, out any) error {
	q.Set("format", "json")
	resp, err := lastfmHTTP.Get("https://ws.audioscrobbler.com/2.0/?" + q.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func LastFMRecent(apiKey, user string) (*Track, error) {
	q := url.Values{}
	q.Set("method", "user.getrecenttracks")
	q.Set("user", user)
	q.Set("api_key", apiKey)
	q.Set("limit", "1")

	var data struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
		Recent  struct {
			Track []struct {
				Name   string `json:"name"`
				URL    string `json:"url"`
				Artist struct {
					Text string `json:"#text"`
				} `json:"artist"`
				Album struct {
					Text string `json:"#text"`
				} `json:"album"`
				Attr struct {
					NowPlaying string `json:"nowplaying"`
				} `json:"@attr"`
			} `json:"track"`
		} `json:"recenttracks"`
	}
	if err := lastfmGet(q, &data); err != nil {
		return nil, err
	}

	switch data.Error {
	case 0:
	case 6:
		return nil, ErrLastFMUserNotFound
	default:
		return nil, fmt.Errorf("last.fm error %d: %s", data.Error, data.Message)
	}

	if len(data.Recent.Track) == 0 {
		return nil, ErrNoTracks
	}
	t := data.Recent.Track[0]
	return &Track{
		Name:       t.Name,
		Artist:     t.Artist.Text,
		Album:      t.Album.Text,
		URL:        t.URL,
		NowPlaying: t.Attr.NowPlaying == "true",
	}, nil
}

func LastFMPlayCount(apiKey, user, artist, track string) (int, error) {
	q := url.Values{}
	q.Set("method", "track.getInfo")
	q.Set("api_key", apiKey)
	q.Set("username", user)
	q.Set("artist", artist)
	q.Set("track", track)
	q.Set("autocorrect", "1")

	var data struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
		Track   struct {
			UserPlayCount string `json:"userplaycount"`
		} `json:"track"`
	}
	if err := lastfmGet(q, &data); err != nil {
		return 0, err
	}
	if data.Error != 0 {
		return 0, fmt.Errorf("last.fm error %d: %s", data.Error, data.Message)
	}
	return strconv.Atoi(data.Track.UserPlayCount)
}
