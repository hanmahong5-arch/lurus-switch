//go:build !windows

package toolconfig

import "os/exec"

// execStart starts a command without waiting for it to complete
func execStart(name string, args ...string) error {
	return exec.Command(name, args...).Start()
}
