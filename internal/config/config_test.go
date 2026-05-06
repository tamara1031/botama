package config

import (
	"testing"
)

func TestLoad_MissingToken(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when DISCORD_TOKEN is empty")
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("MODULES_ENABLED", "")
	t.Setenv("API_ADDR", "")
	t.Setenv("GUILD_ID", "")
	t.Setenv("API_TOKEN", "")
	t.Setenv("NOTIFICATION_CHANNEL_ID", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Token != "tok" {
		t.Errorf("Token: want tok, got %q", cfg.Token)
	}
	if cfg.APIAddr != ":8080" {
		t.Errorf("APIAddr: want :8080, got %q", cfg.APIAddr)
	}
	if len(cfg.EnabledModules) != 0 {
		t.Errorf("EnabledModules: want empty, got %v", cfg.EnabledModules)
	}
}

func TestLoad_ModulesParsed(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("MODULES_ENABLED", "ping, notify , ping")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"ping", "notify", "ping"}
	if len(cfg.EnabledModules) != len(want) {
		t.Fatalf("EnabledModules: want %v, got %v", want, cfg.EnabledModules)
	}
	for i, m := range want {
		if cfg.EnabledModules[i] != m {
			t.Errorf("EnabledModules[%d]: want %q, got %q", i, m, cfg.EnabledModules[i])
		}
	}
}

func TestLoad_CustomAPIAddr(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("API_ADDR", ":9090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIAddr != ":9090" {
		t.Errorf("APIAddr: want :9090, got %q", cfg.APIAddr)
	}
}

func TestLoad_AllFields(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "discord-tok")
	t.Setenv("GUILD_ID", "guild-1")
	t.Setenv("API_TOKEN", "api-tok")
	t.Setenv("API_ADDR", ":7777")
	t.Setenv("NOTIFICATION_CHANNEL_ID", "ch-99")
	t.Setenv("MODULES_ENABLED", "ping,notify")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checks := map[string]string{
		"Token":            cfg.Token,
		"GuildID":          cfg.GuildID,
		"APIToken":         cfg.APIToken,
		"APIAddr":          cfg.APIAddr,
		"DefaultChannelID": cfg.DefaultChannelID,
	}
	want := map[string]string{
		"Token":            "discord-tok",
		"GuildID":          "guild-1",
		"APIToken":         "api-tok",
		"APIAddr":          ":7777",
		"DefaultChannelID": "ch-99",
	}
	for field, got := range checks {
		if got != want[field] {
			t.Errorf("%s: want %q, got %q", field, want[field], got)
		}
	}
	if len(cfg.EnabledModules) != 2 {
		t.Errorf("EnabledModules: want 2 entries, got %v", cfg.EnabledModules)
	}
}
