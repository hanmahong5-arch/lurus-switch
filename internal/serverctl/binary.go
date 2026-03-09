package serverctl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	// githubReleasesURL is the base URL for lurus-newapi GitHub releases.
	githubReleasesURL = "https://github.com/hanmahong5-arch/lurus-newapi/releases/latest/download"

	// binaryName is the executable name for the gateway server on Windows.
	binaryNameWindows = "newapi.exe"
	binaryNameUnix    = "newapi"

	// serverSubDir is the subdirectory within the app data dir for server files.
	serverSubDir = "server"
)

// binaryName returns the platform-appropriate executable name.
func binaryName() string {
	if runtime.GOOS == "windows" {
		return binaryNameWindows
	}
	return binaryNameUnix
}

// detectBinary locates the server binary in priority order:
// 1. Same directory as the running executable (bundled scenario)
// 2. <appDataDir>/server/<binaryName>
// Returns the full path if found and executable, or "" if not found.
func detectBinary(appDataDir string) string {
	candidates := []string{}

	// Priority 1: next to the running executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates, filepath.Join(exeDir, binaryName()))
	}

	// Priority 2: app data server directory
	candidates = append(candidates, filepath.Join(appDataDir, serverSubDir, binaryName()))

	for _, p := range candidates {
		if isExecutable(p) {
			return p
		}
	}
	return ""
}

// isExecutable reports whether path exists and is a regular file.
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// defaultBinaryPath returns the expected destination path for a downloaded binary.
func defaultBinaryPath(appDataDir string) string {
	return filepath.Join(appDataDir, serverSubDir, binaryName())
}

// downloadBinary downloads the lurus-newapi binary from GitHub Releases into destPath.
// progress receives byte counts (downloaded, total) for UI progress updates.
func downloadBinary(ctx context.Context, destPath string, progress func(downloaded, total int64)) error {
	osArch := platformSuffix()
	url := fmt.Sprintf("%s/lurus-newapi-%s", githubReleasesURL, osArch)

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create server directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d from %s", resp.StatusCode, url)
	}

	// Write to a temp file first, then rename atomically.
	tmpPath := destPath + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	var downloaded int64
	total := resp.ContentLength
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				f.Close()
				os.Remove(tmpPath)
				return fmt.Errorf("write binary: %w", writeErr)
			}
			downloaded += int64(n)
			if progress != nil {
				progress(downloaded, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			f.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("read download stream: %w", readErr)
		}
	}
	f.Close()

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("finalize binary: %w", err)
	}
	return nil
}

// platformSuffix returns the OS/arch string used in the release filename.
func platformSuffix() string {
	os_ := runtime.GOOS
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	}
	if os_ == "windows" {
		return fmt.Sprintf("windows-%s.exe", arch)
	}
	return fmt.Sprintf("%s-%s", os_, arch)
}
