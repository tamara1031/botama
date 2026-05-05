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
		EnabledModules: modules,
	}, nil
}
