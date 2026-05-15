package main

import (
	"testing"

	"github.com/tamara1031/botama/internal/config"
)

func TestFactory_Ping_AlwaysSucceeds(t *testing.T) {
	m, err := factories["ping"](&config.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name() != "ping" {
		t.Errorf("Name: want ping, got %q", m.Name())
	}
}

func TestFactory_Notify_MissingToken(t *testing.T) {
	_, err := factories["notify"](&config.Config{})
	if err == nil {
		t.Fatal("expected error when API_TOKEN is missing")
	}
}

func TestFactory_Notify_WithToken(t *testing.T) {
	cfg := &config.Config{Notify: config.NotifyConfig{APIToken: "secret"}}
	m, err := factories["notify"](cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name() != "notify" {
		t.Errorf("Name: want notify, got %q", m.Name())
	}
}

func TestFactories_ContainsKnownModules(t *testing.T) {
	for _, name := range []string{"ping", "notify"} {
		if _, ok := factories[name]; !ok {
			t.Errorf("factory for module %q not registered", name)
		}
	}
}
