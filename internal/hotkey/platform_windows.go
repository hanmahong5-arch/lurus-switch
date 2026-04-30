//go:build windows

package hotkey

import (
	xhotkey "golang.design/x/hotkey"
)

// lookupModifier maps a normalised modifier token to xhotkey.Modifier constants.
// Windows: Cmd / Command / Meta maps to ModWin; ModAlt = Option.
func lookupModifier(upper string) ([]xhotkey.Modifier, bool) {
	switch upper {
	case "CTRL", "CONTROL":
		return []xhotkey.Modifier{xhotkey.ModCtrl}, true
	case "ALT", "OPTION":
		return []xhotkey.Modifier{xhotkey.ModAlt}, true
	case "SHIFT":
		return []xhotkey.Modifier{xhotkey.ModShift}, true
	case "CMD", "COMMAND", "META", "WIN", "SUPER":
		// On Windows the "Windows" modifier key maps to ModWin.
		return []xhotkey.Modifier{xhotkey.ModWin}, true
	case "COMMANDORCONTROL", "CTRLORCMD":
		// Electron-style cross-platform modifier: Ctrl on non-mac.
		return []xhotkey.Modifier{xhotkey.ModCtrl}, true
	}
	return nil, false
}

// platformNamedKeys returns the F-keys and special named keys for Windows.
func platformNamedKeys() map[string]xhotkey.Key {
	return map[string]xhotkey.Key{
		"F1": xhotkey.KeyF1, "F2": xhotkey.KeyF2, "F3": xhotkey.KeyF3,
		"F4": xhotkey.KeyF4, "F5": xhotkey.KeyF5, "F6": xhotkey.KeyF6,
		"F7": xhotkey.KeyF7, "F8": xhotkey.KeyF8, "F9": xhotkey.KeyF9,
		"F10": xhotkey.KeyF10, "F11": xhotkey.KeyF11, "F12": xhotkey.KeyF12,

		"SPACE":     xhotkey.KeySpace,
		"ENTER":     xhotkey.KeyReturn,
		"RETURN":    xhotkey.KeyReturn,
		"TAB":       xhotkey.KeyTab,
		"ESC":       xhotkey.KeyEscape,
		"ESCAPE":    xhotkey.KeyEscape,
		"UP":        xhotkey.KeyUp,
		"DOWN":      xhotkey.KeyDown,
		"LEFT":      xhotkey.KeyLeft,
		"RIGHT":     xhotkey.KeyRight,
		"DELETE": xhotkey.KeyDelete,
		"DEL":    xhotkey.KeyDelete,
	}
}
