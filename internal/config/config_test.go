package config

import (
	"testing"
)

func TestLoad_missingToken(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when DISCORD_TOKEN is empty")
	}
}

func TestLoad_defaults(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("API_ADDR", "")
	t.Setenv("MODULES_ENABLED", "")
	t.Setenv("GUILD_ID", "")
	t.Setenv("API_TOKEN", "")
	t.Setenv("NOTIFICATION_CHANNEL_ID", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIAddr != ":8080" {
		t.Errorf("APIAddr = %q, want :8080", cfg.APIAddr)
	}
	if len(cfg.EnabledModules) != 0 {
		t.Errorf("EnabledModules = %v, want empty", cfg.EnabledModules)
	}
}

func TestLoad_moduleParsing(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("MODULES_ENABLED", " ping , notify , ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"ping", "notify"}
	if len(cfg.EnabledModules) != len(want) {
		t.Fatalf("EnabledModules = %v, want %v", cfg.EnabledModules, want)
	}
	for i, m := range cfg.EnabledModules {
		if m != want[i] {
			t.Errorf("EnabledModules[%d] = %q, want %q", i, m, want[i])
		}
	}
}

func TestLoad_customAddr(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("API_ADDR", ":9090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIAddr != ":9090" {
		t.Errorf("APIAddr = %q, want :9090", cfg.APIAddr)
	}
}
