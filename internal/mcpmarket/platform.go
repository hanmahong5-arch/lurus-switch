package mcpmarket

import "runtime"

// isWindows reports whether the current OS is Windows.
func isWindows() bool {
	return runtime.GOOS == "windows"
}
