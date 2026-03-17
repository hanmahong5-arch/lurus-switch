package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const downloadChunkSize = 64 * 1024 // 64 KB

// Options configures a DownloadFile operation.
type Options struct {
	// ProgressFn is called after each 64 KB chunk is written.
	// downloaded is the running byte total; total is Content-Length (-1 if unknown).
	// percent is 0-100 (0 when total is unknown).
	ProgressFn func(downloaded, total int64, percent int)
}

// DownloadFile downloads url to destPath, invoking opts.ProgressFn every 64 KB.
// The destination directory is created automatically.
// On failure the partial file is removed.
func DownloadFile(ctx context.Context, url, destPath string, opts Options) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed for %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d for %s", resp.StatusCode, url)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}

	total := resp.ContentLength // -1 if unknown
	var downloaded int64
	buf := make([]byte, downloadChunkSize)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				out.Close()
				os.Remove(destPath)
				return fmt.Errorf("write error: %w", writeErr)
			}
			downloaded += int64(n)
			if opts.ProgressFn != nil {
				pct := 0
				if total > 0 {
					pct = int(downloaded * 100 / total)
				}
				opts.ProgressFn(downloaded, total, pct)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			out.Close()
			os.Remove(destPath)
			return fmt.Errorf("read error: %w", readErr)
		}

		select {
		case <-ctx.Done():
			out.Close()
			os.Remove(destPath)
			return ctx.Err()
		default:
		}
	}

	return out.Close()
}

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
