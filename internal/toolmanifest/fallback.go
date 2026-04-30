package toolmanifest

import (
	_ "embed"
	"encoding/json"
	"log"
	"sync"
)

//go:embed manifest_builtin.json
var builtinJSON []byte

var (
	builtinOnce     sync.Once
	builtinManifest *Manifest
)

// Builtin returns the compile-time embedded manifest.
// It is used as the last-resort fallback when network and cache are both unavailable.
// If the embedded JSON fails to parse (build error), logs the failure and returns
// an empty manifest rather than crashing the desktop app.
func Builtin() *Manifest {
	builtinOnce.Do(func() {
		var m Manifest
		if err := json.Unmarshal(builtinJSON, &m); err != nil {
			log.Printf("toolmanifest: corrupt builtin manifest (build error): %v", err)
			builtinManifest = &Manifest{}
			return
		}
		builtinManifest = &m
	})
	return builtinManifest
}
