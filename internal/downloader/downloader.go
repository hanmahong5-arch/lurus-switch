package downloader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

// Downloader handles downloading binaries and packages
type Downloader struct {
	cacheDir string
	client   *http.Client
}

// DownloadResult contains information about a downloaded file
type DownloadResult struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Version string `json:"version"`
}

// NewDownloader creates a new downloader instance
func NewDownloader(cacheDir string) *Downloader {
	return &Downloader{
		cacheDir: cacheDir,
		client:   &http.Client{},
	}
}

// GetCacheDir returns the platform-specific cache directory
func GetCacheDir() (string, error) {
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

// Download downloads a file from the given URL to the cache
func (d *Downloader) Download(url, filename string) (*DownloadResult, error) {
	destPath := filepath.Join(d.cacheDir, filename)

	// Check if already cached
	if info, err := os.Stat(destPath); err == nil {
		return &DownloadResult{
			Path: destPath,
			Size: info.Size(),
		}, nil
	}

	resp, err := d.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	size, err := io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return &DownloadResult{
		Path: destPath,
		Size: size,
	}, nil
}

// FetchJSON fetches JSON data from a URL
func (d *Downloader) FetchJSON(url string, target interface{}) error {
	resp, err := d.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch failed: HTTP %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return nil
}

// ClearCache removes all cached files
func (d *Downloader) ClearCache() error {
	return os.RemoveAll(d.cacheDir)
}
