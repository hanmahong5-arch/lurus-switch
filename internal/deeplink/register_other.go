//go:build !windows

// Non-Windows stub for protocol handler registration.
//
// macOS: URL scheme registration is done via Info.plist / wails.json at build time, not at runtime.
// Add the following to wails.json "mac" section before packaging:
//
//	"info": {
//	  "CFBundleURLTypes": [
//	    {
//	      "CFBundleURLName": "Lurus Switch Protocol",
//	      "CFBundleURLSchemes": ["switch"]
//	    }
//	  ]
//	}
//
// Linux: handled by .desktop file + xdg-mime (out of scope for this implementation).

package deeplink

import (
	"fmt"
	"os/user"
)

// Register is a no-op on non-Windows platforms.
// macOS protocol registration happens via Info.plist at packaging time.
func Register(_ string) error {
	// TODO(macOS): URL scheme registration is handled by Info.plist CFBundleURLSchemes.
	// No runtime action needed.
	return nil
}

// Unregister is a no-op on non-Windows platforms.
func Unregister() error {
	return nil
}

// currentUsername returns the OS username for IPC socket naming.
func currentUsername() string {
	u, err := user.Current()
	if err != nil {
		return "default"
	}
	return u.Username
}

// VerifyRegisterWritten is a test helper — only meaningful on Windows.
// On non-Windows platforms it always returns an error.
func VerifyRegisterWritten(_ string) error {
	return fmt.Errorf("registry verification not available on this platform")
}
