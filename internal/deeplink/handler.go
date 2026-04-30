// handler.go wires deep-link payloads into the Wails event bus.
//
// Usage in app.go OnStartup:
//
//	server.Start(ctx, deeplink.MakeWailsHandler(ctx))
//
// The frontend receives the event "deeplink:import" with a JSON payload:
//
//	{ "type": "provider"|"mcp"|"prompt"|"skill", "data": {...}, "raw": "switch://..." }

package deeplink

import (
	"context"
	"fmt"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	// EventImport is the Wails event name emitted when an import deep-link arrives.
	EventImport = "deeplink:import"
)

// MakeWailsHandler returns an onPayload callback that emits a Wails event for
// each valid Payload.  The ctx must be the Wails startup context.
func MakeWailsHandler(ctx context.Context) func(*Payload) {
	return func(p *Payload) {
		if p == nil {
			return
		}
		wailsRuntime.EventsEmit(ctx, EventImport, map[string]any{
			"type": p.Type,
			"data": p.Data,
			"raw":  p.Raw,
		})
		fmt.Printf("[deeplink] emitted %s event: type=%s\n", EventImport, p.Type)
	}
}
