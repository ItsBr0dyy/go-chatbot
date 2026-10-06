package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

var ErrMCSRNotFound = errors.New("mcsr ranked profile not found")

type RankedCasual struct {
	Ranked *int `json:"ranked"`
	Casual *int `json:"casual"`
}

func (r RankedCasual) R() int {
	if r.Ranked == nil {
		return 0
	}
	return *r.Ranked
}

type MCSRStats struct {
	BestTime         RankedCasual `json:"bestTime"`
	HighestWinStreak RankedCasual `json:"highestWinStreak"`
	PlayedMatches    RankedCasual `json:"playedMatches"`
	CompletionTime   RankedCasual `json:"completionTime"`
	Completions      RankedCasual `json:"completions"`
	Forfeits         RankedCasual `json:"forfeits"`
	Wins             RankedCasual `json:"wins"`
	Loses            RankedCasual `json:"loses"`
}

type MCSRUser struct {
	UUID         string  `json:"uuid"`
	Nickname     string  `json:"nickname"`
	EloRate      *int    `json:"eloRate"`
	EloRank      *int    `json:"eloRank"`
	Country      *string `json:"country"`
	SeasonResult *struct {
		Highest *int `json:"highest"`
		Lowest  *int `json:"lowest"`
	} `json:"seasonResult"`
	Statistics struct {
		Season MCSRStats `json:"season"`
		Total  MCSRStats `json:"total"`
	} `json:"statistics"`
}

var mcsrHTTP = &http.Client{Timeout: 6 * time.Second}

func MCSRLookup(name string) (*MCSRUser, error) {
	resp, err := mcsrHTTP.Get("https://api.mcsrranked.com/users/" + url.PathEscape(name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusBadRequest, http.StatusNotFound:
		return nil, ErrMCSRNotFound
	case http.StatusTooManyRequests:
		return nil, errors.New("rate limited, try again in a bit")
	default:
		return nil, fmt.Errorf("mcsr ranked returned %d", resp.StatusCode)
	}

	var env struct {
		Status string          `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, err
	}
	if env.Status != "success" {
		return nil, ErrMCSRNotFound
	}

	var u MCSRUser
	if err := json.Unmarshal(env.Data, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// FormatMS formats milliseconds as m:ss.mmm.
func FormatMS(ms int) string {
	return fmt.Sprintf("%d:%02d.%03d", ms/60000, (ms%60000)/1000, ms%1000)
}

func FormatMinSec(ms int) string {
	return fmt.Sprintf("%02d:%02d", ms/60000, (ms%60000)/1000)
}

func MCSRTier(elo int) string {
	switch {
	case elo >= 2000:
		return "Netherite"
	case elo >= 1500:
		return "Diamond"
	case elo >= 1200:
		return "Emerald"
	case elo >= 900:
		return "Gold"
	case elo >= 600:
		return "Iron"
	default:
		return "Coal"
	}
}
