package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Username   string   `json:"username"`
	OAuth      string   `json:"oauth"`
	Helpers    []string `json:"helpers"`
	Owners     []string `json:"owners"`
	Prefix     string   `json:"prefix"`
	LastFMKey  string   `json:"lastfm_api_key"`
	SongEmote  string   `json:"song_emote"`
	ShortAPI   string   `json:"short_api"`
	PasteAPI   string   `json:"paste_api"`
	DBHost     string   `json:"db_host"`
	DBPort     string   `json:"db_port"`
	DBUser     string   `json:"db_user"`
	DBPassword string   `json:"db_password"`
	DBName     string   `json:"db_name"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Username == "" || cfg.OAuth == "" {
		return nil, fmt.Errorf("config needs username and oauth")
	}
	if cfg.DBUser == "" || cfg.DBPassword == "" || cfg.DBName == "" {
		return nil, fmt.Errorf("config needs db_user, db_password and db_name")
	}
	if cfg.DBHost == "" {
		cfg.DBHost = "localhost"
	}
	if cfg.DBPort == "" {
		cfg.DBPort = "5432"
	}
	if cfg.Prefix == "" {
		cfg.Prefix = "!"
	}
	if len(cfg.Owners) == 0 {
		cfg.Owners = []string{"itsbr0dyy", "wydios"}
	}
	if cfg.SongEmote == "" {
		cfg.SongEmote = "LUL"
	}
	if cfg.ShortAPI == "" {
		cfg.ShortAPI = "https://itsbr0dyy.dev/api/redirect"
	}
	if cfg.PasteAPI == "" {
		cfg.PasteAPI = "https://itsbr0dyy.dev/api/haste"
	}
	return &cfg, nil
}

func (c *Config) DatabaseURL() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.DBUser, c.DBPassword),
		Host:     net.JoinHostPort(c.DBHost, c.DBPort),
		Path:     "/" + c.DBName,
		RawQuery: "sslmode=disable",
	}
	return u.String()
}

func (c *Config) IsHelper(login string) bool {
	for _, h := range c.Helpers {
		if strings.EqualFold(h, login) {
			return true
		}
	}
	return false
}

func (c *Config) IsOwner(login string) bool {
	for _, o := range c.Owners {
		if strings.EqualFold(o, login) {
			return true
		}
	}
	return false
}
