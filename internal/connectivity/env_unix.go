//go:build !windows

package connectivity

import "os"

func envLookup(k string) string { return os.Getenv(k) }
