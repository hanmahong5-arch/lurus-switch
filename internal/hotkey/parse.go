// Package hotkey provides global hotkey registration and management.
// It wraps golang.design/x/hotkey with configuration persistence and
// a human-readable shortcut string parser.
package hotkey

import (
	"errors"
	"fmt"
	"strings"

	xhotkey "golang.design/x/hotkey"
)

// ErrDisabled is returned when the shortcut string is empty (hotkey intentionally off).
var ErrDisabled = errors.New("hotkey: disabled (empty shortcut)")

// ErrInvalidKey is returned when a token cannot be mapped to a known key or modifier.
var ErrInvalidKey = errors.New("hotkey: invalid key or modifier")

// parsed holds the result of parsing a shortcut string.
type parsed struct {
	mods []xhotkey.Modifier
	key  xhotkey.Key
}

// parseShortcut converts a human-readable shortcut string to modifiers + key.
//
// Supported forms:
//
//	"Ctrl+Shift+S"         → [ModCtrl, ModShift], KeyS
//	"Cmd+Shift+S"          → [ModCmd/ModWin], [ModShift], KeyS
//	"CommandOrControl+Q"   → ModCmd on darwin, ModCtrl elsewhere + KeyQ
//	"  ctrl + shift + s "  → tolerates whitespace and case
//	""                     → ErrDisabled
//	"InvalidKey"           → ErrInvalidKey
func parseShortcut(shortcut string) (parsed, error) {
	shortcut = strings.TrimSpace(shortcut)
	if shortcut == "" {
		return parsed{}, ErrDisabled
	}

	tokens := strings.Split(shortcut, "+")

	var mods []xhotkey.Modifier
	var key xhotkey.Key
	keyFound := false

	for _, raw := range tokens {
		tok := strings.TrimSpace(raw)
		if tok == "" {
			continue
		}
		upper := strings.ToUpper(tok)

		// Try modifier first.
		if mod, ok := lookupModifier(upper); ok {
			mods = append(mods, mod...)
			continue
		}

		// Try key.
		if k, ok := lookupKey(upper); ok {
			if keyFound {
				return parsed{}, fmt.Errorf("%w: multiple non-modifier keys in %q", ErrInvalidKey, shortcut)
			}
			key = k
			keyFound = true
			continue
		}

		return parsed{}, fmt.Errorf("%w: unknown token %q in shortcut %q", ErrInvalidKey, tok, shortcut)
	}

	if !keyFound {
		return parsed{}, fmt.Errorf("%w: no non-modifier key found in %q", ErrInvalidKey, shortcut)
	}

	return parsed{mods: mods, key: key}, nil
}

// lookupKey maps a normalised uppercase token to an xhotkey.Key constant.
func lookupKey(upper string) (xhotkey.Key, bool) {
	k, ok := allKeys[upper]
	return k, ok
}

// allKeys maps normalised key names → Key constants defined by the platform build.
// Letters (A-Z) and digits (0-9) are handled inline below; named keys are in
// the platform-specific files (keys_windows.go / keys_darwin.go / keys_other.go).
var allKeys = buildKeyMap()

func buildKeyMap() map[string]xhotkey.Key {
	m := make(map[string]xhotkey.Key, 64)

	// Letters A-Z
	letters := map[string]xhotkey.Key{
		"A": xhotkey.KeyA, "B": xhotkey.KeyB, "C": xhotkey.KeyC, "D": xhotkey.KeyD,
		"E": xhotkey.KeyE, "F": xhotkey.KeyF, "G": xhotkey.KeyG, "H": xhotkey.KeyH,
		"I": xhotkey.KeyI, "J": xhotkey.KeyJ, "K": xhotkey.KeyK, "L": xhotkey.KeyL,
		"M": xhotkey.KeyM, "N": xhotkey.KeyN, "O": xhotkey.KeyO, "P": xhotkey.KeyP,
		"Q": xhotkey.KeyQ, "R": xhotkey.KeyR, "S": xhotkey.KeyS, "T": xhotkey.KeyT,
		"U": xhotkey.KeyU, "V": xhotkey.KeyV, "W": xhotkey.KeyW, "X": xhotkey.KeyX,
		"Y": xhotkey.KeyY, "Z": xhotkey.KeyZ,
	}
	for k, v := range letters {
		m[k] = v
	}

	// Digits 0-9
	digits := map[string]xhotkey.Key{
		"0": xhotkey.Key0, "1": xhotkey.Key1, "2": xhotkey.Key2, "3": xhotkey.Key3,
		"4": xhotkey.Key4, "5": xhotkey.Key5, "6": xhotkey.Key6, "7": xhotkey.Key7,
		"8": xhotkey.Key8, "9": xhotkey.Key9,
	}
	for k, v := range digits {
		m[k] = v
	}

	// F-keys and named keys — defined per-platform via platformNamedKeys()
	for k, v := range platformNamedKeys() {
		m[k] = v
	}

	return m
}
