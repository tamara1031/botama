package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Token          string
	GuildID        string
	APIToken       string
	APIAddr        string
	NotifyChannels map[string]string
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

	return &Config{
		Token:          token,
		GuildID:        os.Getenv("GUILD_ID"),
		APIToken:       os.Getenv("API_TOKEN"),
		APIAddr:        apiAddr,
		NotifyChannels: loadNotifyChannels(),
		EnabledModules: modules,
	}, nil
}

const (
	notifyEnvPrefix = "NOTIFY_"
	notifyEnvSuffix = "_CHANNEL_ID"
)

// loadNotifyChannels discovers NOTIFY_<LEVEL>_CHANNEL_ID env vars and returns
// them as a level→channelID map. The level is lowercased (e.g. "info", "warning").
func loadNotifyChannels() map[string]string {
	channels := make(map[string]string)
	for _, env := range os.Environ() {
		name, val, hasVal := strings.Cut(env, "=")
		if !hasVal || val == "" {
			continue
		}
		if strings.HasPrefix(name, notifyEnvPrefix) && strings.HasSuffix(name, notifyEnvSuffix) {
			level := strings.ToLower(name[len(notifyEnvPrefix) : len(name)-len(notifyEnvSuffix)])
			if level != "" {
				channels[level] = val
			}
		}
	}
	return channels
}
