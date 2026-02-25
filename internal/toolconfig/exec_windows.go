//go:build windows

package toolconfig

import (
	"os/exec"
	"syscall"
)

// execStart starts a command without waiting for it to complete, hiding the console window
func execStart(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd.Start()
}
