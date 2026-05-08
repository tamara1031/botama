package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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

	b.RegisterModule(ping.New(cfg.GuildID))
	b.RegisterModule(notify.New(cfg.APIToken, notify.Channels(cfg.NotifyChannels), cfg.APIAddr))

	if err := b.Start(); err != nil {
		slog.Error("start", "error", err)
		os.Exit(1)
	}

	slog.Info("running — press Ctrl+C to stop")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	if err := b.Stop(); err != nil {
		slog.Error("shutdown", "error", err)
		os.Exit(1)
	}
}
