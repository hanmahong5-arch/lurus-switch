package toolmanifest

// Manifest is the top-level structure returned by the download-manifest endpoint.
type Manifest struct {
	GeneratedAt string                `json:"generated_at"`
	Tools       map[string]ToolEntry  `json:"tools"`
}

// ToolEntry describes one tool in the manifest.
// Type is one of: "npm" | "binary" | "desktop".
type ToolEntry struct {
	Type          string                   `json:"type"`
	NpmPackage    string                   `json:"npm_package,omitempty"`
	LatestVersion string                   `json:"latest_version"`
	Platforms     map[string]PlatformAsset `json:"platforms,omitempty"`
}

// PlatformAsset holds the download URL and optional SHA-256 checksum for one
// OS/arch combination. Platform key format: "os/arch" (e.g. "windows/amd64").
type PlatformAsset struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256,omitempty"`
}

// GetPlatformURL returns the download URL for toolName on platform.
// platform must be in "os/arch" format as returned by CurrentPlatform().
// Returns an empty string if the tool or platform is not found.
func (m *Manifest) GetPlatformURL(toolName, platform string) string {
	if a := m.GetPlatformAsset(toolName, platform); a != nil {
		return a.URL
	}
	return ""
}

// GetPlatformAsset returns the full PlatformAsset (URL + SHA256) for toolName on platform.
// Returns nil if the tool or platform is not found.
func (m *Manifest) GetPlatformAsset(toolName, platform string) *PlatformAsset {
	if m == nil {
		return nil
	}
	entry, ok := m.Tools[toolName]
	if !ok || entry.Platforms == nil {
		return nil
	}
	asset, ok := entry.Platforms[platform]
	if !ok {
		return nil
	}
	return &asset
}

// GetLatestVersion returns the latest_version string for toolName.
// Returns an empty string if the tool is not found.
func (m *Manifest) GetLatestVersion(toolName string) string {
	if m == nil {
		return ""
	}
	entry, ok := m.Tools[toolName]
	if !ok {
		return ""
	}
	return entry.LatestVersion
}
