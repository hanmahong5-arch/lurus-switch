//go:build darwin

package hotkey

import (
	xhotkey "golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

// platformRegister calls hk.Register() on the OS main thread (macOS requirement).
// The Carbon RegisterEventHotKey API must be called from the thread that owns
// the Carbon event loop.  Wails v2 on macOS runs its own NSApplication main loop,
// so we use mainthread.Call to marshal the call correctly.
func platformRegister(hk *xhotkey.Hotkey) error {
	var err error
	mainthread.Call(func() {
		err = hk.Register()
	})
	return err
}

// platformUnregister calls hk.Unregister() on the OS main thread.
func platformUnregister(hk *xhotkey.Hotkey) error {
	var err error
	mainthread.Call(func() {
		err = hk.Unregister()
	})
	return err
}
