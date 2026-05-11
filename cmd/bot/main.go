package main

import (
	"context"
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

	b.RegisterModule(ping.New(cfg.Discord.GuildID))
	b.RegisterModule(notify.New(notify.Config{
		Token:    cfg.Notify.Token,
		Addr:     cfg.Notify.Addr,
		Channels: notify.Channels(cfg.Notify.Channels),
	}))

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
