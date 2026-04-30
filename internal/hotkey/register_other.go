//go:build !windows && !darwin

package hotkey

import (
	xhotkey "golang.design/x/hotkey"
)

// platformRegister calls hk.Register() on Linux/other platforms.
// Linux requires CGO (libX11); registration can happen on any thread.
func platformRegister(hk *xhotkey.Hotkey) error {
	return hk.Register()
}

// platformUnregister calls hk.Unregister() on Linux/other platforms.
func platformUnregister(hk *xhotkey.Hotkey) error {
	return hk.Unregister()
}
