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
	t.Setenv("GUILD_ID", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Token != "tok" {
		t.Errorf("Token: want tok, got %q", cfg.Token)
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

func TestLoad_AllFields(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "discord-tok")
	t.Setenv("GUILD_ID", "guild-1")
	t.Setenv("MODULES_ENABLED", "ping,notify")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Token != "discord-tok" {
		t.Errorf("Token: want discord-tok, got %q", cfg.Token)
	}
	if cfg.GuildID != "guild-1" {
		t.Errorf("GuildID: want guild-1, got %q", cfg.GuildID)
	}
	if len(cfg.EnabledModules) != 2 {
		t.Errorf("EnabledModules: want 2 entries, got %v", cfg.EnabledModules)
	}
}
