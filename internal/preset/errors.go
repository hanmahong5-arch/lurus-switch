package preset

import "fmt"

// unknownPreset returns a descriptive error for an unrecognized preset ID.
func unknownPreset(tool, id string) error {
	return fmt.Errorf("unknown preset %q for tool %q", id, tool)
}
