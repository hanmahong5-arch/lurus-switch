package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const githubRequestTimeout = 15 * time.Second

// GitHubChecker checks GitHub releases for the latest version
type GitHubChecker struct {
	client *http.Client
	owner  string
	repo   string
}

// githubRelease represents the minimal GitHub release API response
type githubRelease struct {
	TagName string        `json:"tag_name"`
	HTMLURL string        `json:"html_url"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// NewGitHubChecker creates a new GitHubChecker for the given repo
func NewGitHubChecker(owner, repo string) *GitHubChecker {
	return &GitHubChecker{
		client: &http.Client{Timeout: githubRequestTimeout},
		owner:  owner,
		repo:   repo,
	}
}

// CheckUpdate checks GitHub releases for the latest version
func (g *GitHubChecker) CheckUpdate(name, currentVersion string) (*UpdateInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", g.owner, g.repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return &UpdateInfo{
			Name:            name,
			CurrentVersion:  currentVersion,
			LatestVersion:   "unknown (offline)",
			UpdateAvailable: false,
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")

	// Find the download URL for the current platform binary if available
	downloadURL := release.HTMLURL
	for _, asset := range release.Assets {
		if matchesPlatformAsset(asset.Name) {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	return &UpdateInfo{
		Name:            name,
		CurrentVersion:  currentVersion,
		LatestVersion:   latestVersion,
		UpdateAvailable: latestVersion != "" && currentVersion != "" && isNewer(latestVersion, currentVersion),
		DownloadURL:     downloadURL,
	}, nil
}

// matchesPlatformAsset checks if an asset filename matches the current OS/arch
func matchesPlatformAsset(name string) bool {
	lower := strings.ToLower(name)
	// Check for Windows x64
	if strings.Contains(lower, "windows") && (strings.Contains(lower, "x64") || strings.Contains(lower, "amd64")) {
		return true
	}
	return false
}
