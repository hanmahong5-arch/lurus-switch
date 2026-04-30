// Windows implementation of protocol handler registration.
// Writes to HKCU\Software\Classes\switch — no admin rights required.

package deeplink

import (
	"fmt"
	"os/user"

	"golang.org/x/sys/windows/registry"
)

// Register installs the OS URL handler so clicking switch://... launches exePath
// with the URL as argv[1].  Idempotent; safe to call on every startup.
//
// Registry layout (HKCU, no admin required):
//
//	HKCU\Software\Classes\switch
//	  (Default) = "URL:Lurus Switch Protocol"
//	  URL Protocol = ""
//	  \DefaultIcon
//	      (Default) = "<exePath>,0"
//	  \shell\open\command
//	      (Default) = `"<exePath>" "%1"`
func Register(exePath string) error {
	const base = `Software\Classes\switch`

	// Root key.
	k, _, err := registry.CreateKey(registry.CURRENT_USER, base, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("deeplink register: create root key: %w", err)
	}
	defer k.Close()
	if err := k.SetStringValue("", "URL:Lurus Switch Protocol"); err != nil {
		return fmt.Errorf("deeplink register: set default value: %w", err)
	}
	if err := k.SetStringValue("URL Protocol", ""); err != nil {
		return fmt.Errorf("deeplink register: set URL Protocol: %w", err)
	}

	// DefaultIcon sub-key.
	iconKey, _, err := registry.CreateKey(registry.CURRENT_USER, base+`\DefaultIcon`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("deeplink register: create DefaultIcon key: %w", err)
	}
	defer iconKey.Close()
	if err := iconKey.SetStringValue("", exePath+",0"); err != nil {
		return fmt.Errorf("deeplink register: set DefaultIcon: %w", err)
	}

	// shell\open\command sub-key.
	cmdKey, _, err := registry.CreateKey(registry.CURRENT_USER, base+`\shell\open\command`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("deeplink register: create command key: %w", err)
	}
	defer cmdKey.Close()
	cmdValue := fmt.Sprintf(`"%s" "%%1"`, exePath)
	if err := cmdKey.SetStringValue("", cmdValue); err != nil {
		return fmt.Errorf("deeplink register: set command value: %w", err)
	}

	return nil
}

// Unregister removes the OS URL handler from the registry. Best-effort.
// Deletes sub-keys in dependency order before removing the root key.
func Unregister() error {
	// Sub-keys must be removed before their parent.
	subKeys := []string{
		`Software\Classes\switch\shell\open\command`,
		`Software\Classes\switch\shell\open`,
		`Software\Classes\switch\shell`,
		`Software\Classes\switch\DefaultIcon`,
		`Software\Classes\switch`,
	}
	var lastErr error
	for _, sk := range subKeys {
		if err := registry.DeleteKey(registry.CURRENT_USER, sk); err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		return fmt.Errorf("deeplink unregister: %w", lastErr)
	}
	return nil
}

// VerifyRegisterWritten reads back the registry command value and checks it
// matches the expected exePath.  Used only in tests.
func VerifyRegisterWritten(exePath string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Classes\switch\shell\open\command`, registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("open command key: %w", err)
	}
	defer k.Close()
	val, _, err := k.GetStringValue("")
	if err != nil {
		return fmt.Errorf("read command value: %w", err)
	}
	want := fmt.Sprintf(`"%s" "%%1"`, exePath)
	if val != want {
		return fmt.Errorf("command value = %q, want %q", val, want)
	}
	return nil
}

// currentUsername returns the local Windows username for pipe naming.
func currentUsername() string {
	u, err := user.Current()
	if err != nil {
		return "default"
	}
	// Strip domain prefix (DOMAIN\User → User).
	name := u.Username
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '\\' || name[i] == '/' {
			return name[i+1:]
		}
	}
	return name
}
