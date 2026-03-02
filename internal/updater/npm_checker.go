package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"lurus-switch/internal/installer"
)

const npmRequestTimeout = 15 * time.Second

// NpmChecker checks npm registry for latest versions of CLI tools
type NpmChecker struct {
	client *http.Client
}

// NewNpmChecker creates a new NpmChecker
func NewNpmChecker() *NpmChecker {
	return &NpmChecker{
		client: &http.Client{Timeout: npmRequestTimeout},
	}
}

// npmPackageInfo represents the minimal npm registry response we need
type npmPackageInfo struct {
	Version string `json:"version"`
}

// CheckUpdate checks the npm registry for the latest version of a package
func (n *NpmChecker) CheckUpdate(packageName, currentVersion string) (*UpdateInfo, error) {
	url := fmt.Sprintf("%s/%s/latest", installer.NpmRegistryURL, packageName)

	resp, err := n.client.Get(url)
	if err != nil {
		return &UpdateInfo{
			Name:            packageName,
			CurrentVersion:  currentVersion,
			LatestVersion:   "unknown (offline)",
			UpdateAvailable: false,
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry returned HTTP %d for package %s", resp.StatusCode, packageName)
	}

	var info npmPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse npm registry response: %w", err)
	}

	return &UpdateInfo{
		Name:            packageName,
		CurrentVersion:  currentVersion,
		LatestVersion:   info.Version,
		UpdateAvailable: info.Version != "" && currentVersion != "" && info.Version != currentVersion && isNewer(info.Version, currentVersion),
	}, nil
}

// CheckAllTools checks updates for all three CLI tools given their current versions
func (n *NpmChecker) CheckAllTools(toolVersions map[string]string) map[string]*UpdateInfo {
	packages := map[string]string{
		installer.ToolClaude: installer.ClaudeNpmPackage,
		installer.ToolCodex:  installer.CodexNpmPackage,
		installer.ToolGemini: installer.GeminiNpmPackage,
	}

	results := make(map[string]*UpdateInfo)
	for toolName, pkgName := range packages {
		currentVersion := toolVersions[toolName]
		info, err := n.CheckUpdate(pkgName, currentVersion)
		if err != nil {
			results[toolName] = &UpdateInfo{
				Name:            toolName,
				CurrentVersion:  currentVersion,
				LatestVersion:   "unknown",
				UpdateAvailable: false,
			}
			continue
		}
		info.Name = toolName
		results[toolName] = info
	}

	return results
}

// IsNewerVersion is the exported version of isNewer for use across packages.
func IsNewerVersion(latest, current string) bool {
	return isNewer(latest, current)
}

// isNewer compares two semver strings and returns true if latest > current
func isNewer(latest, current string) bool {
	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	for i := 0; i < len(latestParts) && i < len(currentParts); i++ {
		l := parseIntSafe(latestParts[i])
		c := parseIntSafe(currentParts[i])
		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}

	return len(latestParts) > len(currentParts)
}

// parseIntSafe parses a string as int, returning 0 on failure
func parseIntSafe(s string) int {
	n := 0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	return n
}
