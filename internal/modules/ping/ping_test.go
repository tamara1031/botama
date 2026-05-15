package ping

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestPing_Name(t *testing.T) {
	p := New("")
	if p.Name() != "ping" {
		t.Errorf("Name: want ping, got %q", p.Name())
	}
}

func TestPing_New_WithGuildID(t *testing.T) {
	p := New("guild-123")
	if p.guildID != "guild-123" {
		t.Errorf("guildID: want guild-123, got %q", p.guildID)
	}
}

// Shutdown before Register must be a no-op: no panic, no error.
func TestPing_Shutdown_BeforeRegister(t *testing.T) {
	p := New("guild-1")
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown before Register: unexpected error: %v", err)
	}
}

// interactionUserID returns the Member's user ID when Member is set.
func TestInteractionUserID_MemberPresent(t *testing.T) {
	i := &discordgo.Interaction{
		Member: &discordgo.Member{User: &discordgo.User{ID: "member-id"}},
	}
	if got := interactionUserID(i); got != "member-id" {
		t.Errorf("want member-id, got %q", got)
	}
}

// interactionUserID falls back to User when Member is nil (DM context).
func TestInteractionUserID_UserFallback(t *testing.T) {
	i := &discordgo.Interaction{
		User: &discordgo.User{ID: "dm-user"},
	}
	if got := interactionUserID(i); got != "dm-user" {
		t.Errorf("want dm-user, got %q", got)
	}
}

// Member takes priority over User when both are set.
func TestInteractionUserID_MemberTakesPriority(t *testing.T) {
	i := &discordgo.Interaction{
		Member: &discordgo.Member{User: &discordgo.User{ID: "member"}},
		User:   &discordgo.User{ID: "direct"},
	}
	if got := interactionUserID(i); got != "member" {
		t.Errorf("want member, got %q", got)
	}
}

// Neither Member nor User returns an empty string (no panic).
func TestInteractionUserID_NeitherPresent(t *testing.T) {
	i := &discordgo.Interaction{}
	if got := interactionUserID(i); got != "" {
		t.Errorf("want empty string, got %q", got)
	}
}
