package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Token                  string
	GuildID                string
	APIToken               string
	APIAddr                string
	NotifyInfoChannelID    string
	NotifyWarningChannelID string
	NotifyCriticalChannelID string
	EnabledModules         []string
}

func Load() (*Config, error) {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN is required")
	}

	var modules []string
	seen := make(map[string]bool)
	if raw := os.Getenv("MODULES_ENABLED"); raw != "" {
		for _, m := range strings.Split(raw, ",") {
			if m = strings.TrimSpace(m); m != "" && !seen[m] {
				seen[m] = true
				modules = append(modules, m)
			}
		}
	}

	apiAddr := os.Getenv("API_ADDR")
	if apiAddr == "" {
		apiAddr = ":8080"
	}

	return &Config{
		Token:                   token,
		GuildID:                 os.Getenv("GUILD_ID"),
		APIToken:                os.Getenv("API_TOKEN"),
		APIAddr:                 apiAddr,
		NotifyInfoChannelID:     os.Getenv("NOTIFY_INFO_CHANNEL_ID"),
		NotifyWarningChannelID:  os.Getenv("NOTIFY_WARNING_CHANNEL_ID"),
		NotifyCriticalChannelID: os.Getenv("NOTIFY_CRITICAL_CHANNEL_ID"),
		EnabledModules:          modules,
	}, nil
}
