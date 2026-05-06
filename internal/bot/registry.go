package bot

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type registry struct {
	modules map[string]Module
	active  map[string]bool
	order   []string // registration order; stopAll iterates in reverse (LIFO)
}

func newRegistry() *registry {
	return &registry{
		modules: make(map[string]Module),
		active:  make(map[string]bool),
	}
}

func (r *registry) add(m Module) {
	name := m.Name()
	if _, exists := r.modules[name]; !exists {
		r.order = append(r.order, name)
	}
	r.modules[name] = m
}

func (r *registry) startEnabled(s *discordgo.Session, names []string) error {
	for _, name := range names {
		m, ok := r.modules[name]
		if !ok {
			return fmt.Errorf("module %q is not registered", name)
		}
		if err := m.Register(s); err != nil {
			return fmt.Errorf("module %q: %w", name, err)
		}
		r.active[name] = true
	}
	return nil
}

func (r *registry) stopAll() error {
	var errs []error
	for i := len(r.order) - 1; i >= 0; i-- {
		name := r.order[i]
		if !r.active[name] {
			continue
		}
		if err := r.modules[name].Unregister(); err != nil {
			errs = append(errs, fmt.Errorf("module %q unregister: %w", name, err))
		}
		r.active[name] = false
	}
	return errors.Join(errs...)
}
