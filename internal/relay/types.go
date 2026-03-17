package relay

// RelayKind classifies the origin of a relay endpoint.
type RelayKind string

const (
	KindLurus      RelayKind = "lurus"
	KindThirdParty RelayKind = "third_party"
	KindCustom     RelayKind = "custom"
)

// RelayEndpoint represents a single API relay endpoint configuration.
type RelayEndpoint struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Kind        RelayKind `json:"kind"`
	URL         string    `json:"url"`
	APIKey      string    `json:"apiKey"`
	Description string    `json:"description,omitempty"`

	// Runtime fields — populated by health checks, not persisted.
	LatencyMs   int64  `json:"latencyMs"`
	Healthy     bool   `json:"healthy"`
	LastChecked string `json:"lastChecked,omitempty"`
}

// ToolRelayMapping maps a tool name to the ID of its preferred relay endpoint.
type ToolRelayMapping map[string]string
