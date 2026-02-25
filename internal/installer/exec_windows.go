//go:build windows

package installer

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// hideWindow prevents a console window from flashing when running a subprocess
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
