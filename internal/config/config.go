package config

import (
	"fmt"
	"os"
	"strings"
)

type DiscordConfig struct {
	Token   string
	GuildID string
}

type NotifyConfig struct {
	APIToken        string
	APIAddr         string
	InfoChannel     string
	WarningChannel  string
	CriticalChannel string
}

type Config struct {
	Discord        DiscordConfig
	Notify         NotifyConfig
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
		Discord: DiscordConfig{
			Token:   token,
			GuildID: os.Getenv("GUILD_ID"),
		},
		Notify: NotifyConfig{
			APIToken:        os.Getenv("API_TOKEN"),
			APIAddr:         apiAddr,
			InfoChannel:     os.Getenv("NOTIFY_INFO_CHANNEL_ID"),
			WarningChannel:  os.Getenv("NOTIFY_WARNING_CHANNEL_ID"),
			CriticalChannel: os.Getenv("NOTIFY_CRITICAL_CHANNEL_ID"),
		},
		EnabledModules: modules,
	}, nil
}
