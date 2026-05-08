package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tamara1031/botama/internal/bot"
	"github.com/tamara1031/botama/internal/config"
	"github.com/tamara1031/botama/internal/modules/notify"
	"github.com/tamara1031/botama/internal/modules/ping"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}

	b, err := bot.New(cfg)
	if err != nil {
		slog.Error("init", "error", err)
		os.Exit(1)
	}

	b.RegisterModule(ping.New(cfg.Discord.GuildID))
	b.RegisterModule(notify.New(cfg.Notify.APIToken, notify.Channels{
		Info:     cfg.Notify.InfoChannel,
		Warning:  cfg.Notify.WarningChannel,
		Critical: cfg.Notify.CriticalChannel,
	}, cfg.Notify.APIAddr))

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
