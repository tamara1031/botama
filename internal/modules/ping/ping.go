package ping

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

var command = &discordgo.ApplicationCommand{
	Name:        "ping",
	Description: "Botの疎通確認",
}

type Ping struct {
	guildID       string
	session       *discordgo.Session
	removeHandler func()
	commandID     string
}

func New(guildID string) *Ping {
	return &Ping{guildID: guildID}
}

func (p *Ping) Name() string { return "ping" }

func (p *Ping) Register(s *discordgo.Session) error {
	cmd, err := s.ApplicationCommandCreate(s.State.User.ID, p.guildID, command)
	if err != nil {
		return fmt.Errorf("ping: register slash command: %w", err)
	}
	p.session = s
	p.commandID = cmd.ID

	scope := "global"
	if p.guildID != "" {
		scope = "guild:" + p.guildID
	}
	slog.Info("ping module registered", "command_id", cmd.ID, "scope", scope)

	p.removeHandler = s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		slog.Debug("interaction received", "type", i.Type, "guild", i.GuildID)
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}
		if i.ApplicationCommandData().Name != "ping" {
			return
		}
		userID := interactionUserID(i.Interaction)
		slog.Info("ping: received", "user", userID, "guild", i.GuildID, "channel", i.ChannelID)
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "pong"},
		}); err != nil {
			slog.Error("ping: failed to respond", "error", err)
			return
		}
		slog.Info("ping: responded", "user", userID)
	})
	return nil
}

func interactionUserID(i *discordgo.Interaction) string {
	if i.Member != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func (p *Ping) Unregister() error {
	if p.removeHandler != nil {
		p.removeHandler()
	}
	if p.commandID != "" {
		if err := p.session.ApplicationCommandDelete(p.session.State.User.ID, p.guildID, p.commandID); err != nil {
			slog.Warn("ping: failed to delete slash command", "error", err)
		}
	}
	slog.Info("ping module unregistered")
	return nil
}
