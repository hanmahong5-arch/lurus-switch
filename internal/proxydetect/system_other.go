//go:build !windows

package proxydetect

// detectSystemProxy is a no-op on non-Windows platforms.
// macOS/Linux typically use environment variables which are already handled by detectEnvVars.
func detectSystemProxy() []DetectedProxy {
	return nil
}
