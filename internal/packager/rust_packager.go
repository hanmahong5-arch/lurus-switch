package packager

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// RustPackager handles downloading and packaging Codex CLI binaries
type RustPackager struct {
	cacheDir string
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

// GitHubAsset represents a release asset
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// NewRustPackager creates a new Rust packager
func NewRustPackager() (*RustPackager, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, err
	}
	return &RustPackager{cacheDir: cacheDir}, nil
}

// getCacheDir returns the cache directory for downloaded binaries
func getCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	var cacheDir string
	switch runtime.GOOS {
	case "windows":
		cacheDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "lurus-switch", "cache")
	case "darwin":
		cacheDir = filepath.Join(home, "Library", "Caches", "lurus-switch")
	default:
		xdgCache := os.Getenv("XDG_CACHE_HOME")
		if xdgCache == "" {
			xdgCache = filepath.Join(home, ".cache")
		}
		cacheDir = filepath.Join(xdgCache, "lurus-switch")
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	return cacheDir, nil
}

// DownloadCodex downloads the Codex CLI binary from GitHub
func (p *RustPackager) DownloadCodex(version string) (string, error) {
	// Get platform-specific asset name
	assetName := p.getAssetName()
	if assetName == "" {
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Check cache first
	cachedPath := filepath.Join(p.cacheDir, "codex", version, assetName)
	if _, err := os.Stat(cachedPath); err == nil {
		return cachedPath, nil
	}

	// Fetch release info
	releaseURL := "https://api.github.com/repos/openai/codex/releases/latest"
	if version != "latest" {
		releaseURL = fmt.Sprintf("https://api.github.com/repos/openai/codex/releases/tags/%s", version)
	}

	resp, err := http.Get(releaseURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch release info: HTTP %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse release info: %w", err)
	}

	// Find matching asset
	var downloadURL string
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, assetName) || p.matchesAsset(asset.Name) {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return "", fmt.Errorf("no matching asset found for platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Download the binary
	if err := os.MkdirAll(filepath.Dir(cachedPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := p.downloadFile(downloadURL, cachedPath); err != nil {
		return "", err
	}

	// Make executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(cachedPath, 0755); err != nil {
			return "", fmt.Errorf("failed to make binary executable: %w", err)
		}
	}

	return cachedPath, nil
}

// getAssetName returns the expected asset name for the current platform
func (p *RustPackager) getAssetName() string {
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "amd64" {
			return "codex-windows-x64.exe"
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "codex-darwin-arm64"
		}
		return "codex-darwin-x64"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "codex-linux-arm64"
		}
		return "codex-linux-x64"
	}
	return ""
}

// matchesAsset checks if an asset name matches the current platform
func (p *RustPackager) matchesAsset(name string) bool {
	name = strings.ToLower(name)

	// Check OS
	switch runtime.GOOS {
	case "windows":
		if !strings.Contains(name, "windows") && !strings.Contains(name, "win") {
			return false
		}
	case "darwin":
		if !strings.Contains(name, "darwin") && !strings.Contains(name, "macos") && !strings.Contains(name, "apple") {
			return false
		}
	case "linux":
		if !strings.Contains(name, "linux") {
			return false
		}
	default:
		return false
	}

	// Check arch
	switch runtime.GOARCH {
	case "amd64":
		return strings.Contains(name, "x64") || strings.Contains(name, "amd64") || strings.Contains(name, "x86_64")
	case "arm64":
		return strings.Contains(name, "arm64") || strings.Contains(name, "aarch64")
	}

	return false
}

// downloadFile downloads a file from URL to the given path
func (p *RustPackager) downloadFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Package creates a packaged Codex CLI with custom configuration
func (p *RustPackager) Package(configDir, outputDir string, version string) (string, error) {
	// Download the binary
	binaryPath, err := p.DownloadCodex(version)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Copy the binary to output directory
	outputPath := filepath.Join(outputDir, filepath.Base(binaryPath))
	if err := p.copyFile(binaryPath, outputPath); err != nil {
		return "", err
	}

	return outputPath, nil
}

// copyFile copies a file from src to dst
func (p *RustPackager) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Preserve file permissions on Unix
	if runtime.GOOS != "windows" {
		srcInfo, err := os.Stat(src)
		if err == nil {
			os.Chmod(dst, srcInfo.Mode())
		}
	}

	return nil
}

// GetCacheDir returns the cache directory path
func (p *RustPackager) GetCacheDir() string {
	return p.cacheDir
}
