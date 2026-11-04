package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrRolesNotFound = errors.New("no roles data found")

type RolesStats struct {
	Total      int
	Partners   int
	Affiliates int
	Followers  int
}

var rolesHTTP = &http.Client{Timeout: 8 * time.Second}

const maxRolesPages = 10

func RolesTVStats(kind, login string) (*RolesStats, error) {
	stats := &RolesStats{}

	for page := 1; page <= maxRolesPages; page++ {
		u := fmt.Sprintf(
			"https://roles.tv/api/stats/user/%s/login/%s?per_page=100&include_inactive=false&page=%d",
			kind, url.PathEscape(login), page)

		resp, err := rolesHTTP.Get(u)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return nil, ErrRolesNotFound
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("roles.tv returned %d", resp.StatusCode)
		}

		var data struct {
			Total int `json:"total"`
			Page  int `json:"page"`
			Pages int `json:"pages"`
			Data  []struct {
				Followers   int  `json:"followers"`
				IsPartner   bool `json:"isPartner"`
				IsAffiliate bool `json:"isAffiliate"`
			} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&data)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if page > 1 && data.Page != page {
			break
		}

		if page == 1 {
			stats.Total = data.Total
		}
		for _, c := range data.Data {
			stats.Followers += c.Followers
			if c.IsPartner {
				stats.Partners++
			}
			if c.IsAffiliate {
				stats.Affiliates++
			}
		}

		if page >= data.Pages {
			break
		}
	}
	return stats, nil
}

func FormatInt(n int) string {
	s := strconv.Itoa(n)
	if n < 1000 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
