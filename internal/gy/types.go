package gy

// ProductKind describes how a GY product is launched.
type ProductKind string

const (
	KindWeb     ProductKind = "web"     // Open in browser
	KindDesktop ProductKind = "desktop" // Launch local executable
	KindService ProductKind = "service" // Open service dashboard in browser
)

// GYProduct defines a GY suite product.
type GYProduct struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Kind        ProductKind `json:"kind"`

	// web / service
	LaunchURL string `json:"launchUrl,omitempty"`
	// desktop
	DownloadURL string `json:"downloadUrl,omitempty"`
	// service dashboard URL (may differ from internal URL)
	ServiceURL string `json:"serviceUrl,omitempty"`
}

// GYStatus holds runtime availability info for a product.
type GYStatus struct {
	ProductID string `json:"productId"`
	Available bool   `json:"available"`
	LatencyMs int64  `json:"latencyMs"`
	// Version is populated for desktop products when the exe is found locally.
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}
