//go:build windows

package connectivity

import "os"

// envLookup is identical to the Unix implementation; the split exists so a
// future enhancement that reads the Windows registry (HKCU\Software\Microsoft
// \Windows\CurrentVersion\Internet Settings\ProxyServer) can drop in without
// affecting other platforms.
func envLookup(k string) string { return os.Getenv(k) }
