package bot

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// mockModule is a Module that records calls and returns configurable errors.
type mockModule struct {
	name        string
	registered  bool
	unregistered bool
	registerErr  error
	unregErr     error
}

func (m *mockModule) Name() string { return m.name }

func (m *mockModule) Register(_ *discordgo.Session) error {
	m.registered = true
	return m.registerErr
}

func (m *mockModule) Unregister() error {
	m.unregistered = true
	return m.unregErr
}

// --- add / startEnabled ---

func TestRegistry_StartEnabled_UnknownModule(t *testing.T) {
	r := newRegistry()
	err := r.startEnabled(nil, []string{"ghost"})
	if err == nil {
		t.Fatal("expected error for unregistered module")
	}
}

func TestRegistry_StartEnabled_RegisterError(t *testing.T) {
	r := newRegistry()
	m := &mockModule{name: "bad", registerErr: errors.New("boom")}
	r.add(m)

	err := r.startEnabled(nil, []string{"bad"})
	if err == nil {
		t.Fatal("expected error from Register")
	}
	if r.active["bad"] {
		t.Error("module must not be marked active when Register returns an error")
	}
}

func TestRegistry_StartEnabled_Success(t *testing.T) {
	r := newRegistry()
	m := &mockModule{name: "ok"}
	r.add(m)

	if err := r.startEnabled(nil, []string{"ok"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.registered {
		t.Error("expected Register to be called")
	}
	if !r.active["ok"] {
		t.Error("expected module to be marked active")
	}
}

func TestRegistry_StartEnabled_EmptyList(t *testing.T) {
	r := newRegistry()
	r.add(&mockModule{name: "unused"})

	if err := r.startEnabled(nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- stopAll ---

func TestRegistry_StopAll_OnlyStopsActive(t *testing.T) {
	r := newRegistry()
	active := &mockModule{name: "active"}
	inactive := &mockModule{name: "inactive"}
	r.add(active)
	r.add(inactive)
	_ = r.startEnabled(nil, []string{"active"})

	if err := r.stopAll(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !active.unregistered {
		t.Error("active module should have been unregistered")
	}
	if inactive.unregistered {
		t.Error("inactive module must not be unregistered")
	}
}

func TestRegistry_StopAll_ReturnsError(t *testing.T) {
	r := newRegistry()
	m := &mockModule{name: "fail", unregErr: errors.New("unregister failed")}
	r.add(m)
	_ = r.startEnabled(nil, []string{"fail"})

	err := r.stopAll()
	if err == nil {
		t.Fatal("expected error from Unregister")
	}
}

func TestRegistry_StopAll_ReturnsAllErrors(t *testing.T) {
	r := newRegistry()
	m1 := &mockModule{name: "fail1", unregErr: errors.New("err1")}
	m2 := &mockModule{name: "fail2", unregErr: errors.New("err2")}
	r.add(m1)
	r.add(m2)
	_ = r.startEnabled(nil, []string{"fail1", "fail2"})

	err := r.stopAll()
	if err == nil {
		t.Fatal("expected errors from both modules")
	}
	if !errors.Is(err, m1.unregErr) {
		t.Errorf("expected err1 in joined error, got: %v", err)
	}
	if !errors.Is(err, m2.unregErr) {
		t.Errorf("expected err2 in joined error, got: %v", err)
	}
}

func TestRegistry_StopAll_MarksInactive(t *testing.T) {
	r := newRegistry()
	m := &mockModule{name: "m"}
	r.add(m)
	_ = r.startEnabled(nil, []string{"m"})
	_ = r.stopAll()

	if r.active["m"] {
		t.Error("module should be marked inactive after stopAll")
	}
}

func TestRegistry_Add_OverwritesSameName(t *testing.T) {
	r := newRegistry()
	first := &mockModule{name: "dup"}
	second := &mockModule{name: "dup"}
	r.add(first)
	r.add(second)

	_ = r.startEnabled(nil, []string{"dup"})
	if !second.registered {
		t.Error("second registration should win")
	}
	if first.registered {
		t.Error("first registration should have been overwritten")
	}
}
