//go:build windows

package process

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// hideWindowProcess prevents a console window from appearing for subprocess on Windows
func hideWindowProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
