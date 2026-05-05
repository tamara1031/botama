package ping

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

type Ping struct {
	removeHandler func()
}

func New() *Ping {
	return &Ping{}
}

func (p *Ping) Name() string { return "ping" }

func (p *Ping) Register(s *discordgo.Session) error {
	p.removeHandler = s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}
		if m.Content == "!ping" {
			if _, err := s.ChannelMessageSend(m.ChannelID, "pong"); err != nil {
				slog.Error("ping: failed to send pong", "error", err, "channel", m.ChannelID)
			}
		}
	})
	slog.Info("ping module registered")
	return nil
}

func (p *Ping) Unregister() error {
	if p.removeHandler != nil {
		p.removeHandler()
	}
	slog.Info("ping module unregistered")
	return nil
}
