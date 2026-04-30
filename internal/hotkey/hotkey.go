// Package hotkey provides global hotkey registration and management for the
// lurus-switch desktop application.  It wraps golang.design/x/hotkey and
// adds configuration persistence, a human-readable shortcut parser, and a
// clean callback-based API that is decoupled from the window/runtime layer.
package hotkey

import (
	"context"
	"fmt"
	"sync"

	xhotkey "golang.design/x/hotkey"
)

// RegistrationError records a binding that could not be registered.
type RegistrationError struct {
	Binding  string
	Shortcut string
	Err      error
}

func (e RegistrationError) Error() string {
	return fmt.Sprintf("hotkey: failed to register %q (%s): %v", e.Binding, e.Shortcut, e.Err)
}

// Manager manages global hotkey registration and lifecycle.
type Manager struct {
	mu        sync.Mutex
	configDir string
	onTrigger func(binding string)
	bindings  Bindings
	active    map[string]*entry // binding key → active hotkey entry
	stopCh    chan struct{}
	running   bool
}

// entry holds a registered hotkey and its cancel function.
type entry struct {
	hk      *xhotkey.Hotkey
	binding string
	cancel  context.CancelFunc
}

// New creates the manager.
//
//   - configDir is the directory where hotkey.json lives (e.g. %APPDATA%\lurus-switch).
//   - onTrigger is called when any bound hotkey is pressed; binding is the config key
//     (e.g. "quickSwitch"). The callback is invoked from a dedicated goroutine, not
//     the OS main thread, so it is safe to call Wails runtime functions from it.
func New(configDir string, onTrigger func(binding string)) *Manager {
	return &Manager{
		configDir: configDir,
		onTrigger: onTrigger,
		active:    make(map[string]*entry),
	}
}

// Start registers all enabled hotkeys and begins listening. Non-blocking — each
// hotkey runs in its own goroutine.  Returns the list of bindings that failed to
// register (e.g. already held by another application).
func (m *Manager) Start(ctx context.Context) []RegistrationError {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	b, err := loadBindings(m.configDir)
	if err != nil {
		// Non-fatal: use whatever was loaded (defaults on parse failure).
		fmt.Printf("hotkey: config load warning: %v\n", err)
	}
	m.bindings = b
	m.stopCh = make(chan struct{})
	m.running = true

	var errs []RegistrationError
	for k, shortcut := range b {
		if shortcut == "" {
			continue // explicitly disabled
		}
		if regErr := m.registerLocked(ctx, k, shortcut); regErr != nil {
			errs = append(errs, RegistrationError{
				Binding:  k,
				Shortcut: shortcut,
				Err:      regErr,
			})
		}
	}
	return errs
}

// Stop unregisters all hotkeys. Idempotent.
func (m *Manager) Stop() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	close(m.stopCh)

	for k := range m.active {
		m.unregisterLocked(k)
	}
}

// GetBindings returns a copy of the current bindings config.
func (m *Manager) GetBindings() Bindings {
	if m == nil {
		return DefaultBindings()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.bindings == nil {
		return DefaultBindings()
	}

	out := make(Bindings, len(m.bindings))
	for k, v := range m.bindings {
		out[k] = v
	}
	return out
}

// UpdateBinding persists and re-registers a single binding.
// An empty shortcut disables the binding.
func (m *Manager) UpdateBinding(key, shortcut string) error {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.bindings == nil {
		m.bindings = DefaultBindings()
	}

	// Unregister old binding if active.
	m.unregisterLocked(key)

	m.bindings[key] = shortcut

	if err := saveBindings(m.configDir, m.bindings); err != nil {
		return err
	}

	// Re-register if manager is running and shortcut is non-empty.
	if m.running && shortcut != "" {
		if regErr := m.registerLocked(context.Background(), key, shortcut); regErr != nil {
			return RegistrationError{Binding: key, Shortcut: shortcut, Err: regErr}
		}
	}
	return nil
}

// registerLocked parses and registers a single hotkey. Must be called with m.mu held.
func (m *Manager) registerLocked(ctx context.Context, binding, shortcut string) error {
	p, err := parseShortcut(shortcut)
	if err != nil {
		return err
	}

	hk := xhotkey.New(p.mods, p.key)
	if regErr := platformRegister(hk); regErr != nil {
		return regErr
	}

	hkCtx, cancel := context.WithCancel(ctx)
	e := &entry{hk: hk, binding: binding, cancel: cancel}
	m.active[binding] = e

	trigger := m.onTrigger
	stopCh := m.stopCh

	go func() {
		defer cancel()
		for {
			select {
			case _, ok := <-hk.Keydown():
				if !ok {
					return
				}
				if trigger != nil {
					trigger(binding)
				}
			case <-hkCtx.Done():
				return
			case <-stopCh:
				return
			}
		}
	}()

	return nil
}

// unregisterLocked tears down a single active hotkey. Must be called with m.mu held.
// No-op if the binding is not active.
func (m *Manager) unregisterLocked(key string) {
	e, ok := m.active[key]
	if !ok {
		return
	}
	e.cancel()
	// Unregister on the OS layer; ignore error (already unregistered is fine).
	_ = platformUnregister(e.hk)
	delete(m.active, key)
}
