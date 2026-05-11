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
	t.Setenv("NOTIFY_INFO_CHANNEL_ID", "")
	t.Setenv("NOTIFY_WARNING_CHANNEL_ID", "")
	t.Setenv("NOTIFY_CRITICAL_CHANNEL_ID", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("LOG_FORMAT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Discord.Token != "tok" {
		t.Errorf("Discord.Token: want tok, got %q", cfg.Discord.Token)
	}
	if cfg.Notify.Addr != ":8080" {
		t.Errorf("Notify.Addr: want :8080, got %q", cfg.Notify.Addr)
	}
	if len(cfg.EnabledModules) != 0 {
		t.Errorf("EnabledModules: want empty, got %v", cfg.EnabledModules)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel: want info, got %q", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat: want json, got %q", cfg.LogFormat)
	}
}

func TestLoad_LogLevelAndFormat(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel: want debug, got %q", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat: want text, got %q", cfg.LogFormat)
	}
}

func TestLoad_ModulesParsed(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("MODULES_ENABLED", "ping, notify , ping")
	// notify module requires API_TOKEN
	t.Setenv("API_TOKEN", "required-for-notify")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// duplicate "ping" must be removed; order of first occurrence is preserved
	want := []string{"ping", "notify"}
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
	if cfg.Notify.Addr != ":9090" {
		t.Errorf("Notify.Addr: want :9090, got %q", cfg.Notify.Addr)
	}
}

func TestLoad_AllFields(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "discord-tok")
	t.Setenv("GUILD_ID", "guild-1")
	t.Setenv("API_TOKEN", "api-tok")
	t.Setenv("API_ADDR", ":7777")
	t.Setenv("NOTIFY_INFO_CHANNEL_ID", "ch-info")
	t.Setenv("NOTIFY_WARNING_CHANNEL_ID", "ch-warn")
	t.Setenv("NOTIFY_CRITICAL_CHANNEL_ID", "ch-crit")
	t.Setenv("MODULES_ENABLED", "ping,notify")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Discord.Token != "discord-tok" {
		t.Errorf("Discord.Token: want discord-tok, got %q", cfg.Discord.Token)
	}
	if cfg.Discord.GuildID != "guild-1" {
		t.Errorf("Discord.GuildID: want guild-1, got %q", cfg.Discord.GuildID)
	}
	if cfg.Notify.Token != "api-tok" {
		t.Errorf("Notify.Token: want api-tok, got %q", cfg.Notify.Token)
	}
	if cfg.Notify.Addr != ":7777" {
		t.Errorf("Notify.Addr: want :7777, got %q", cfg.Notify.Addr)
	}
	if cfg.Notify.Channels["info"] != "ch-info" {
		t.Errorf("Notify.Channels[info]: want ch-info, got %q", cfg.Notify.Channels["info"])
	}
	if cfg.Notify.Channels["warning"] != "ch-warn" {
		t.Errorf("Notify.Channels[warning]: want ch-warn, got %q", cfg.Notify.Channels["warning"])
	}
	if cfg.Notify.Channels["critical"] != "ch-crit" {
		t.Errorf("Notify.Channels[critical]: want ch-crit, got %q", cfg.Notify.Channels["critical"])
	}

	if len(cfg.EnabledModules) != 2 {
		t.Errorf("EnabledModules: want 2 entries, got %v", cfg.EnabledModules)
	}
}

// --- validate ---

func TestLoad_NotifyModuleRequiresAPIToken(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("MODULES_ENABLED", "notify")
	t.Setenv("API_TOKEN", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when notify is enabled without API_TOKEN")
	}
}

func TestLoad_NotifyModuleWithAPIToken(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("MODULES_ENABLED", "notify")
	t.Setenv("API_TOKEN", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Notify.Token != "secret" {
		t.Errorf("Notify.Token: want secret, got %q", cfg.Notify.Token)
	}
}

func TestLoad_PingModuleNoExtraReqs(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("MODULES_ENABLED", "ping")
	t.Setenv("API_TOKEN", "")

	_, err := Load()
	if err != nil {
		t.Fatalf("ping module should not require API_TOKEN, got: %v", err)
	}
}

func TestLoad_NotifyChannelsDynamic(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "tok")
	t.Setenv("NOTIFY_INFO_CHANNEL_ID", "ch-info")
	t.Setenv("NOTIFY_WARNING_CHANNEL_ID", "ch-warn")
	t.Setenv("NOTIFY_EMERGENCY_CHANNEL_ID", "ch-emergency")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cases := []struct{ level, want string }{
		{"info", "ch-info"},
		{"warning", "ch-warn"},
		{"emergency", "ch-emergency"},
	}
	for _, tc := range cases {
		if got := cfg.Notify.Channels[tc.level]; got != tc.want {
			t.Errorf("Channels[%q]: want %q, got %q", tc.level, tc.want, got)
		}
	}
}
