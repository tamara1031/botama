package ping

import (
	"context"
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// fakeSession implements discordSession for testing without a real Discord connection.
type fakeSession struct {
	createErr error
	createID  string

	deleteErr    error
	deleteCalled bool

	respondErr    error
	respondCalled bool
	lastResponse  *discordgo.InteractionResponse
}

func (f *fakeSession) ApplicationCommandCreate(_, _ string, cmd *discordgo.ApplicationCommand, _ ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	id := f.createID
	if id == "" {
		id = "cmd-id-123"
	}
	out := *cmd
	out.ID = id
	return &out, nil
}

func (f *fakeSession) ApplicationCommandDelete(_, _, _ string, _ ...discordgo.RequestOption) error {
	f.deleteCalled = true
	return f.deleteErr
}

func (f *fakeSession) AddHandler(_ interface{}) func() {
	return func() {}
}

func (f *fakeSession) InteractionRespond(_ *discordgo.Interaction, r *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
	f.respondCalled = true
	f.lastResponse = r
	return f.respondErr
}

// newRegisteredPing creates a Ping with register() already called against fakeSession.
func newRegisteredPing(t *testing.T, fake *fakeSession) *Ping {
	t.Helper()
	p := New("guild-id")
	if err := p.register(fake, "app-id"); err != nil {
		t.Fatalf("register: %v", err)
	}
	return p
}

// pingInteraction creates a well-formed /ping InteractionCreate for tests.
func pingInteraction() *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{Name: "ping"},
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "user-1"},
			},
		},
	}
}

// --- Name ---

func TestName(t *testing.T) {
	if got := New("").Name(); got != "ping" {
		t.Errorf("Name() = %q, want %q", got, "ping")
	}
}

// --- interactionUserID ---

func TestInteractionUserID_Member(t *testing.T) {
	i := &discordgo.Interaction{Member: &discordgo.Member{User: &discordgo.User{ID: "member-user"}}}
	if got := interactionUserID(i); got != "member-user" {
		t.Errorf("got %q, want %q", got, "member-user")
	}
}

func TestInteractionUserID_UserFallback(t *testing.T) {
	i := &discordgo.Interaction{User: &discordgo.User{ID: "direct-user"}}
	if got := interactionUserID(i); got != "direct-user" {
		t.Errorf("got %q, want %q", got, "direct-user")
	}
}

func TestInteractionUserID_MemberPriority(t *testing.T) {
	i := &discordgo.Interaction{
		Member: &discordgo.Member{User: &discordgo.User{ID: "member"}},
		User:   &discordgo.User{ID: "top-level"},
	}
	if got := interactionUserID(i); got != "member" {
		t.Errorf("got %q, want %q", got, "member")
	}
}

func TestInteractionUserID_Empty(t *testing.T) {
	i := &discordgo.Interaction{}
	if got := interactionUserID(i); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

// --- register ---

func TestRegister_GuildScope(t *testing.T) {
	fake := &fakeSession{}
	p := New("guild-123")

	if err := p.register(fake, "app-id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.commandID != "cmd-id-123" {
		t.Errorf("commandID = %q, want %q", p.commandID, "cmd-id-123")
	}
	if p.session == nil {
		t.Error("session not stored")
	}
	if p.appID != "app-id" {
		t.Errorf("appID = %q, want %q", p.appID, "app-id")
	}
	if p.removeHandler == nil {
		t.Error("removeHandler not set")
	}
}

func TestRegister_GlobalScope(t *testing.T) {
	fake := &fakeSession{}
	p := New("") // empty guildID → global scope

	if err := p.register(fake, "app-id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.commandID == "" {
		t.Error("commandID should be set for global scope")
	}
}

func TestRegister_CreateError(t *testing.T) {
	want := errors.New("discord API error")
	fake := &fakeSession{createErr: want}
	p := New("")

	err := p.register(fake, "app-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, want) {
		t.Errorf("got %v, want wrapped %v", err, want)
	}
}

// --- handleInteraction ---

func TestHandleInteraction_PingCommand_Responds(t *testing.T) {
	fake := &fakeSession{}
	p := newRegisteredPing(t, fake)

	p.handleInteraction(pingInteraction())

	if !fake.respondCalled {
		t.Fatal("InteractionRespond not called")
	}
	if fake.lastResponse.Data.Content != "pong" {
		t.Errorf("response content = %q, want %q", fake.lastResponse.Data.Content, "pong")
	}
	if fake.lastResponse.Type != discordgo.InteractionResponseChannelMessageWithSource {
		t.Errorf("response type = %v, want ChannelMessageWithSource", fake.lastResponse.Type)
	}
}

func TestHandleInteraction_NonCommandType_Ignored(t *testing.T) {
	fake := &fakeSession{}
	p := newRegisteredPing(t, fake)

	p.handleInteraction(&discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent, // not ApplicationCommand
		},
	})

	if fake.respondCalled {
		t.Error("InteractionRespond should not be called for non-command interactions")
	}
}

func TestHandleInteraction_OtherCommand_Ignored(t *testing.T) {
	fake := &fakeSession{}
	p := newRegisteredPing(t, fake)

	p.handleInteraction(&discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{Name: "other"},
		},
	})

	if fake.respondCalled {
		t.Error("InteractionRespond should not be called for other commands")
	}
}

func TestHandleInteraction_RespondError_NoReturn(t *testing.T) {
	fake := &fakeSession{respondErr: errors.New("respond failed")}
	p := newRegisteredPing(t, fake)

	// Must not panic; error is logged and handler returns gracefully.
	p.handleInteraction(pingInteraction())

	if !fake.respondCalled {
		t.Error("InteractionRespond should have been attempted")
	}
}

// --- Shutdown ---

func TestShutdown_CallsRemoveHandler(t *testing.T) {
	removed := false
	fake := &fakeSession{}
	p := New("")
	if err := p.register(fake, "app-id"); err != nil {
		t.Fatal(err)
	}
	p.removeHandler = func() { removed = true }

	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Error("removeHandler not called during Shutdown")
	}
}

func TestShutdown_DeletesCommand(t *testing.T) {
	fake := &fakeSession{}
	p := newRegisteredPing(t, fake)

	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !fake.deleteCalled {
		t.Error("ApplicationCommandDelete not called")
	}
}

func TestShutdown_DeleteError_ReturnsNil(t *testing.T) {
	fake := &fakeSession{deleteErr: errors.New("delete failed")}
	p := newRegisteredPing(t, fake)

	// Shutdown warns on delete failure but returns nil — Discord state is advisory.
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown should return nil on delete error, got: %v", err)
	}
}

func TestShutdown_EmptyCommandID_SkipsDelete(t *testing.T) {
	fake := &fakeSession{}
	p := &Ping{session: fake} // commandID intentionally empty

	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fake.deleteCalled {
		t.Error("ApplicationCommandDelete should not be called with empty commandID")
	}
}
