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
	APIToken string
	APIAddr  string
	Channels map[string]string
}

type Config struct {
	Discord        DiscordConfig
	Notify         NotifyConfig
	EnabledModules []string
	LogLevel       string
	LogFormat      string
}

const (
	notifyEnvPrefix = "NOTIFY_"
	notifyEnvSuffix = "_CHANNEL_ID"
)

// loadNotifyChannels discovers NOTIFY_<LEVEL>_CHANNEL_ID env vars at startup.
// Adding a new notification level requires only a new env var — no code change.
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

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "json"
	}

	cfg := &Config{
		Discord: DiscordConfig{
			Token:   token,
			GuildID: os.Getenv("GUILD_ID"),
		},
		Notify: NotifyConfig{
			APIToken: os.Getenv("API_TOKEN"),
			APIAddr:  apiAddr,
			Channels: loadNotifyChannels(),
		},
		EnabledModules: modules,
		LogLevel:       logLevel,
		LogFormat:      logFormat,
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate checks module-specific requirements so misconfiguration is caught
// at startup before the Discord session is opened.
func (c *Config) validate() error {
	for _, name := range c.EnabledModules {
		switch name {
		case "notify":
			if c.Notify.APIToken == "" {
				return fmt.Errorf("module %q requires API_TOKEN", name)
			}
		}
	}
	return nil
}
