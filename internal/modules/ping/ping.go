package ping

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

var command = &discordgo.ApplicationCommand{
	Name:        "ping",
	Description: "Botの疎通確認",
}

// discordSession is the narrow interface of discordgo.Session operations this module requires.
// Depending on this interface instead of the concrete type makes the module independently testable.
type discordSession interface {
	ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
	ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error
	AddHandler(handler interface{}) func()
	InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
}

type Ping struct {
	guildID       string
	appID         string
	session       discordSession
	removeHandler func()
	commandID     string
}

func New(guildID string) *Ping {
	return &Ping{guildID: guildID}
}

func (p *Ping) Name() string { return "ping" }

// Register implements bot.Module. It extracts the app ID from the concrete session
// and delegates to the testable register method.
func (p *Ping) Register(s *discordgo.Session) error {
	return p.register(s, s.State.User.ID)
}

func (p *Ping) register(s discordSession, appID string) error {
	cmd, err := s.ApplicationCommandCreate(appID, p.guildID, command)
	if err != nil {
		return fmt.Errorf("ping: register slash command: %w", err)
	}
	p.session = s
	p.appID = appID
	p.commandID = cmd.ID

	scope := "global"
	if p.guildID != "" {
		scope = "guild:" + p.guildID
	}
	slog.Info("ping module registered", "command_id", cmd.ID, "scope", scope)

	p.removeHandler = s.AddHandler(func(_ *discordgo.Session, i *discordgo.InteractionCreate) {
		p.handleInteraction(i)
	})
	return nil
}

func (p *Ping) handleInteraction(i *discordgo.InteractionCreate) {
	slog.Debug("interaction received", "type", i.Type, "guild", i.GuildID)
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	if i.ApplicationCommandData().Name != "ping" {
		return
	}
	userID := interactionUserID(i.Interaction)
	slog.Info("ping: received", "user", userID, "guild", i.GuildID, "channel", i.ChannelID)
	if err := p.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "pong"},
	}); err != nil {
		slog.Error("ping: failed to respond", "error", err)
		return
	}
	slog.Info("ping: responded", "user", userID)
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

func (p *Ping) Shutdown(_ context.Context) error {
	if p.removeHandler != nil {
		p.removeHandler()
	}
	if p.commandID != "" {
		if err := p.session.ApplicationCommandDelete(p.appID, p.guildID, p.commandID); err != nil {
			slog.Warn("ping: failed to delete slash command", "error", err)
		}
	}
	slog.Info("ping module unregistered")
	return nil
}
