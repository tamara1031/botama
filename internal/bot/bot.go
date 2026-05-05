package bot

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/tamara1031/botama/internal/config"
)

type Bot struct {
	session  *discordgo.Session
	registry *registry
	cfg      *config.Config
}

func New(cfg *config.Config) (*Bot, error) {
	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	s.Identify.Intents = discordgo.IntentsNone

	return &Bot{
		session:  s,
		registry: newRegistry(),
		cfg:      cfg,
	}, nil
}

func (b *Bot) RegisterModule(m Module) {
	b.registry.add(m)
	slog.Debug("module registered", "name", m.Name())
}

func (b *Bot) Start() error {
	ready := make(chan struct{})
	remove := b.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		slog.Info("connected", "username", r.User.Username, "id", r.User.ID)
		close(ready)
	})
	defer remove()

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("open session: %w", err)
	}
	<-ready

	if err := b.registry.startEnabled(b.session, b.cfg.EnabledModules); err != nil {
		_ = b.session.Close()
		return fmt.Errorf("start modules: %w", err)
	}
	slog.Info("bot started", "modules", b.cfg.EnabledModules)
	return nil
}

func (b *Bot) Stop() error {
	if err := b.registry.stopAll(); err != nil {
		slog.Warn("error stopping modules", "error", err)
	}
	return b.session.Close()
}
