package bot

import "github.com/bwmarrin/discordgo"

// Module is the interface all bot modules must implement.
type Module interface {
	// Name returns the unique identifier used in MODULES_ENABLED.
	Name() string
	// Register attaches event handlers to the Discord session.
	Register(s *discordgo.Session) error
	// Unregister removes all event handlers attached by this module.
	Unregister() error
}
