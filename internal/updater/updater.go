package updater

// UpdateInfo represents version update information for a tool or the app itself
type UpdateInfo struct {
	Name             string `json:"name"`
	CurrentVersion   string `json:"currentVersion"`
	LatestVersion    string `json:"latestVersion"`
	UpdateAvailable  bool   `json:"updateAvailable"`
	DownloadURL      string `json:"downloadUrl,omitempty"`
}

// Checker defines the interface for checking updates
type Checker interface {
	// CheckUpdate compares current vs latest version and returns update info
	CheckUpdate(packageName, currentVersion string) (*UpdateInfo, error)
}
