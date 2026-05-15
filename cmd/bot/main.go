package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tamara1031/botama/internal/bot"
	"github.com/tamara1031/botama/internal/config"
	"github.com/tamara1031/botama/internal/modules/notify"
	"github.com/tamara1031/botama/internal/modules/ping"
)

// moduleFactory creates a bot.Module from the global config.
// The factory is responsible for validating its own config prerequisites
// and returning a descriptive error if they are not met.
type moduleFactory func(cfg *config.Config) (bot.Module, error)

// factories maps each known module name to its factory function.
// To add a new module: add one entry here. No other file needs to change.
var factories = map[string]moduleFactory{
	"ping": func(cfg *config.Config) (bot.Module, error) {
		return ping.New(cfg.Discord.GuildID), nil
	},
	"notify": func(cfg *config.Config) (bot.Module, error) {
		if cfg.Notify.APIToken == "" {
			return nil, fmt.Errorf("module %q requires API_TOKEN", "notify")
		}
		return notify.New(notify.LoadConfig()), nil
	},
}

// registerModules instantiates and registers each enabled module using the
// factory registry. Unknown module names are caught here before Start().
func registerModules(b *bot.Bot, cfg *config.Config) error {
	for _, name := range cfg.EnabledModules {
		factory, ok := factories[name]
		if !ok {
			return fmt.Errorf("unknown module %q", name)
		}
		m, err := factory(cfg)
		if err != nil {
			return err
		}
		b.RegisterModule(m)
	}
	return nil
}

func initLogger(level, format string) {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}
	var h slog.Handler
	if strings.ToLower(format) == "text" {
		h = slog.NewTextHandler(os.Stderr, opts)
	} else {
		h = slog.NewJSONHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(h))
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}

	initLogger(cfg.LogLevel, cfg.LogFormat)

	b, err := bot.New(cfg)
	if err != nil {
		slog.Error("init", "error", err)
		os.Exit(1)
	}

	if err := registerModules(b, cfg); err != nil {
		slog.Error("module setup", "error", err)
		os.Exit(1)
	}

	if err := b.Start(); err != nil {
		slog.Error("start", "error", err)
		os.Exit(1)
	}

	slog.Info("running — press Ctrl+C to stop")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := b.Stop(shutdownCtx); err != nil {
		slog.Error("shutdown", "error", err)
		os.Exit(1)
	}
}
