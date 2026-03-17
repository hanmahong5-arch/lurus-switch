package toolmanifest

import (
	_ "embed"
	"encoding/json"
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
func Builtin() *Manifest {
	builtinOnce.Do(func() {
		var m Manifest
		if err := json.Unmarshal(builtinJSON, &m); err != nil {
			// Embedded JSON must always be valid; panic here is a programming error.
			panic("toolmanifest: corrupt builtin manifest: " + err.Error())
		}
		builtinManifest = &m
	})
	return builtinManifest
}
