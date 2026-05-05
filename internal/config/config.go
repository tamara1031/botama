package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Token          string
	GuildID        string
	NotifyChannels map[string]string // name → Discord channel ID
	APIToken       string
	APIAddr        string
	EnabledModules []string
}

func Load() (*Config, error) {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN is required")
	}

	var modules []string
	if raw := os.Getenv("MODULES_ENABLED"); raw != "" {
		for _, m := range strings.Split(raw, ",") {
			if m = strings.TrimSpace(m); m != "" {
				modules = append(modules, m)
			}
		}
	}

	apiAddr := os.Getenv("API_ADDR")
	if apiAddr == "" {
		apiAddr = ":8080"
	}

	channels := parseChannels(os.Getenv("NOTIFY_CHANNELS"))

	// Backward compat: NOTIFICATION_CHANNEL_ID becomes the "default" channel.
	if legacy := os.Getenv("NOTIFICATION_CHANNEL_ID"); legacy != "" {
		if _, exists := channels["default"]; !exists {
			channels["default"] = legacy
		}
	}

	return &Config{
		Token:          token,
		GuildID:        os.Getenv("GUILD_ID"),
		NotifyChannels: channels,
		APIToken:       os.Getenv("API_TOKEN"),
		APIAddr:        apiAddr,
		EnabledModules: modules,
	}, nil
}

// parseChannels parses "name=id,name=id,..." into a map.
// Malformed entries are silently skipped.
func parseChannels(raw string) map[string]string {
	out := make(map[string]string)
	if raw == "" {
		return out
	}
	for _, entry := range strings.Split(raw, ",") {
		name, id, ok := strings.Cut(strings.TrimSpace(entry), "=")
		if !ok || strings.TrimSpace(name) == "" || strings.TrimSpace(id) == "" {
			continue
		}
		out[strings.TrimSpace(name)] = strings.TrimSpace(id)
	}
	return out
}
