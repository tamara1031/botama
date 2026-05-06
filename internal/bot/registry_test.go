package bot

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// fakeModule is a test double that records Register/Unregister calls.
type fakeModule struct {
	name         string
	registerErr  error
	unregisterErr error
	registered   bool
	unregistered bool
}

func (f *fakeModule) Name() string { return f.name }

func (f *fakeModule) Register(_ *discordgo.Session) error {
	f.registered = true
	return f.registerErr
}

func (f *fakeModule) Unregister() error {
	f.unregistered = true
	return f.unregisterErr
}

func TestRegistry_startEnabled_unknownModule(t *testing.T) {
	r := newRegistry()
	err := r.startEnabled(nil, []string{"missing"})
	if err == nil {
		t.Fatal("expected error for unregistered module")
	}
}

func TestRegistry_startEnabled_success(t *testing.T) {
	r := newRegistry()
	m := &fakeModule{name: "ping"}
	r.add(m)

	if err := r.startEnabled(nil, []string{"ping"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.registered {
		t.Error("module.Register was not called")
	}
	if !r.active["ping"] {
		t.Error("module not marked active after start")
	}
}

func TestRegistry_startEnabled_registerError(t *testing.T) {
	r := newRegistry()
	want := errors.New("register failed")
	m := &fakeModule{name: "bad", registerErr: want}
	r.add(m)

	err := r.startEnabled(nil, []string{"bad"})
	if err == nil {
		t.Fatal("expected error from Register")
	}
	if !errors.Is(err, want) {
		t.Errorf("error = %v, want to wrap %v", err, want)
	}
}

func TestRegistry_stopAll_collecstAllErrors(t *testing.T) {
	r := newRegistry()
	errA := errors.New("err-a")
	errB := errors.New("err-b")
	a := &fakeModule{name: "a", unregisterErr: errA}
	b := &fakeModule{name: "b", unregisterErr: errB}
	r.add(a)
	r.add(b)
	r.active["a"] = true
	r.active["b"] = true

	err := r.stopAll()
	if err == nil {
		t.Fatal("expected combined error from stopAll")
	}
	if !errors.Is(err, errA) {
		t.Errorf("combined error does not wrap errA: %v", err)
	}
	if !errors.Is(err, errB) {
		t.Errorf("combined error does not wrap errB: %v", err)
	}
}

func TestRegistry_stopAll_skipsInactive(t *testing.T) {
	r := newRegistry()
	m := &fakeModule{name: "idle"}
	r.add(m)
	// never activated

	if err := r.stopAll(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.unregistered {
		t.Error("Unregister called on inactive module")
	}
}
