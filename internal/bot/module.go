package bot

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

// Module is the interface all bot modules must implement.
type Module interface {
	// Name returns the unique identifier used in MODULES_ENABLED.
	Name() string
	// Register attaches event handlers to the Discord session.
	Register(s *discordgo.Session) error
	// Shutdown tears down the module, respecting ctx for any blocking cleanup.
	Shutdown(ctx context.Context) error
}
