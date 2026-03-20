package installer

import "strings"

// ToolEndpoint adapts a gateway base URL for a specific tool's SDK expectations.
//
// Claude/Gemini/ZeroClaw/OpenClaw SDKs append their own paths (e.g. /v1/messages),
// so they need the bare domain URL.
//
// Codex/PicoClaw/NullClaw SDKs expect a URL ending in /v1 (then append /chat/completions).
func ToolEndpoint(tool, baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	switch tool {
	case ToolCodex, ToolPicoClaw, ToolNullClaw:
		if !strings.HasSuffix(base, "/v1") {
			return base + "/v1"
		}
		return base
	default:
		// Claude, Gemini, ZeroClaw, OpenClaw — strip /v1 if present
		return strings.TrimSuffix(base, "/v1")
	}
}
