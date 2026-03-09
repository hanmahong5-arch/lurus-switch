//go:build !windows

package process

import "os/exec"

// hideWindowProcess is a no-op on non-Windows platforms
func hideWindowProcess(_ *exec.Cmd) {}
