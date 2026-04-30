//go:build windows

package hotkey

import (
	xhotkey "golang.design/x/hotkey"
)

// platformRegister calls hk.Register() on Windows.
// Windows hotkeys use RegisterHotKey via a dedicated goroutine; no main-thread
// requirement — the library handles the message-pump thread internally.
func platformRegister(hk *xhotkey.Hotkey) error {
	return hk.Register()
}

// platformUnregister calls hk.Unregister() on Windows.
func platformUnregister(hk *xhotkey.Hotkey) error {
	return hk.Unregister()
}
