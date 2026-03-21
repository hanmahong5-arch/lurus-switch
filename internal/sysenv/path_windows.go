//go:build windows

package sysenv

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	envRegistryKey = `Environment`
	pathSeparator  = ";"
)

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	procSendMessageTimeout = user32.NewProc("SendMessageTimeoutW")
)

// broadcastSettingChange notifies other processes that environment variables have changed.
// Sends WM_SETTINGCHANGE to all top-level windows with "Environment" parameter.
func broadcastSettingChange() {
	env, _ := syscall.UTF16PtrFromString("Environment")
	// HWND_BROADCAST = 0xFFFF, WM_SETTINGCHANGE = 0x001A
	// SMTO_ABORTIFHUNG = 0x0002, timeout 5000ms
	var result uintptr
	procSendMessageTimeout.Call(
		uintptr(0xFFFF),  // HWND_BROADCAST
		uintptr(0x001A),  // WM_SETTINGCHANGE
		0,
		uintptr(unsafe.Pointer(env)),
		uintptr(0x0002),  // SMTO_ABORTIFHUNG
		uintptr(5000),
		uintptr(unsafe.Pointer(&result)),
	)
}

// readUserPath reads the current user PATH from the registry.
func readUserPath() (string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, envRegistryKey, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("failed to open HKCU\\%s: %w", envRegistryKey, err)
	}
	defer k.Close()

	val, _, err := k.GetStringValue("Path")
	if err != nil {
		if err == registry.ErrNotExist {
			return "", nil
		}
		return "", fmt.Errorf("failed to read HKCU\\%s\\Path: %w", envRegistryKey, err)
	}
	return val, nil
}

// writeUserPath writes the user PATH to the registry and broadcasts the change.
func writeUserPath(path string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, envRegistryKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open HKCU\\%s for writing: %w", envRegistryKey, err)
	}
	defer k.Close()

	// Use REG_EXPAND_SZ so %%USERPROFILE%% etc. are expanded by the shell.
	if err := k.SetExpandStringValue("Path", path); err != nil {
		return fmt.Errorf("failed to write HKCU\\%s\\Path: %w", envRegistryKey, err)
	}
	broadcastSettingChange()
	return nil
}

// GetUserPath returns the user's PATH entries from the Windows registry.
func GetUserPath() ([]PathEntry, error) {
	raw, err := readUserPath()
	if err != nil {
		return nil, err
	}
	return ParsePathEntries(raw, pathSeparator), nil
}

// AddToUserPath appends a directory to the user PATH if not already present.
// Saves a rollback entry before modifying.
func AddToUserPath(dir string) error {
	if dir == "" {
		return fmt.Errorf("directory must not be empty")
	}

	oldPath, err := readUserPath()
	if err != nil {
		return err
	}

	// Check if already present (case-insensitive on Windows).
	dirLower := strings.ToLower(strings.TrimRight(dir, `\/`))
	for _, existing := range strings.Split(oldPath, pathSeparator) {
		if strings.ToLower(strings.TrimRight(strings.TrimSpace(existing), `\/`)) == dirLower {
			return nil // already in PATH
		}
	}

	// Save rollback before modifying.
	if err := SaveRollback(RollbackEntry{
		Action:   "path_add",
		OldValue: oldPath,
		NewValue: dir,
	}); err != nil {
		return fmt.Errorf("failed to save rollback: %w", err)
	}

	newPath := oldPath
	if newPath != "" && !strings.HasSuffix(newPath, pathSeparator) {
		newPath += pathSeparator
	}
	newPath += dir

	return writeUserPath(newPath)
}

// RemoveFromUserPath removes a directory from the user PATH.
// Saves a rollback entry before modifying.
func RemoveFromUserPath(dir string) error {
	if dir == "" {
		return fmt.Errorf("directory must not be empty")
	}

	oldPath, err := readUserPath()
	if err != nil {
		return err
	}

	dirLower := strings.ToLower(strings.TrimRight(dir, `\/`))
	parts := strings.Split(oldPath, pathSeparator)
	var filtered []string
	found := false
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if strings.ToLower(strings.TrimRight(trimmed, `\/`)) == dirLower {
			found = true
			continue
		}
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}

	if !found {
		return nil // not in PATH, nothing to do
	}

	if err := SaveRollback(RollbackEntry{
		Action:   "path_remove",
		OldValue: dir,
		NewValue: oldPath,
	}); err != nil {
		return fmt.Errorf("failed to save rollback: %w", err)
	}

	return writeUserPath(strings.Join(filtered, pathSeparator))
}
