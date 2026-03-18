package installer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"lurus-switch/internal/downloader"
)

// BinaryToolConfig describes a binary tool that can be downloaded from GitHub Releases
type BinaryToolConfig struct {
	Name         string // tool name (e.g. "picoclaw")
	GitHubOwner  string
	GitHubRepo   string
	BinaryName   string // executable name without extension (e.g. "pclaw")
	DefaultModel string // default model for proxy config
}

// GitHubRelease represents the relevant fields of a GitHub release API response
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

// GitHubAsset represents a single asset in a GitHub release
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// fetchLatestRelease fetches the latest release info from GitHub
func fetchLatestRelease(ctx context.Context, owner, repo string) (*GitHubRelease, error) {
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}
	return &release, nil
}

// findPlatformAsset returns the download URL and asset name for the current OS/arch
func findPlatformAsset(assets []GitHubAsset) (downloadURL, assetName string) {
	var osKeywords []string
	var archKeywords []string
	var exts []string

	switch runtime.GOOS {
	case "windows":
		osKeywords = []string{"x86_64-pc-windows-msvc", "windows-x64", "windows-amd64", "windows"}
		exts = []string{".zip", ".exe"}
	case "darwin":
		osKeywords = []string{"darwin", "macos", "apple"}
		if runtime.GOARCH == "arm64" {
			archKeywords = []string{"aarch64", "arm64"}
		} else {
			archKeywords = []string{"x86_64", "amd64", "x64"}
		}
		exts = []string{".tar.gz", ".zip", ""}
	default: // linux
		osKeywords = []string{"linux"}
		if runtime.GOARCH == "arm64" {
			archKeywords = []string{"aarch64", "arm64"}
		} else {
			archKeywords = []string{"x86_64", "amd64", "x64"}
		}
		exts = []string{".tar.gz", ".zip", ""}
	}

	// Priority search: OS keyword + arch keyword + extension
	for _, osKw := range osKeywords {
		for _, a := range assets {
			lower := strings.ToLower(a.Name)
			if !strings.Contains(lower, strings.ToLower(osKw)) {
				continue
			}
			// If arch keywords are specified, require at least one
			archMatch := len(archKeywords) == 0
			for _, ak := range archKeywords {
				if strings.Contains(lower, strings.ToLower(ak)) {
					archMatch = true
					break
				}
			}
			if !archMatch {
				continue
			}
			for _, ext := range exts {
				if ext == "" || strings.HasSuffix(lower, ext) {
					return a.BrowserDownloadURL, a.Name
				}
			}
		}
	}

	// Fallback: any asset with OS-matching keyword
	for _, a := range assets {
		lower := strings.ToLower(a.Name)
		for _, osKw := range osKeywords {
			if strings.Contains(lower, strings.ToLower(osKw)) {
				for _, ext := range exts {
					if ext == "" || strings.HasSuffix(lower, ext) {
						return a.BrowserDownloadURL, a.Name
					}
				}
			}
		}
	}

	return "", ""
}

// extractFromZip extracts the first matching binary from a zip archive to destPath
func extractFromZip(zipPath, binaryName, destPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.EqualFold(filepath.Base(f.Name), binaryName) {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("failed to open zip entry: %w", err)
			}
			defer rc.Close()

			out, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer out.Close()

			if _, err := io.Copy(out, rc); err != nil {
				return fmt.Errorf("failed to extract binary: %w", err)
			}
			if runtime.GOOS != "windows" {
				_ = os.Chmod(destPath, 0755)
			}
			return nil
		}
	}
	return fmt.Errorf("binary %s not found in zip archive", binaryName)
}

// extractFromTarGz extracts the first matching binary from a .tar.gz archive to destPath
func extractFromTarGz(archivePath, binaryName, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open tar.gz: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}
		// Match by base filename (case-insensitive)
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if !strings.EqualFold(filepath.Base(hdr.Name), binaryName) {
			continue
		}
		out, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return fmt.Errorf("failed to extract binary: %w", err)
		}
		if err := out.Close(); err != nil {
			return fmt.Errorf("failed to close output file: %w", err)
		}
		if runtime.GOOS != "windows" {
			_ = os.Chmod(destPath, 0755)
		}
		return nil
	}
	return fmt.Errorf("binary %s not found in tar.gz archive", binaryName)
}

// copyBinaryFile copies a file from src to dst, setting executable permissions on Unix
func copyBinaryFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(dst, 0755)
	}
	return nil
}

// managedBinDir returns the managed bin directory for lurus-switch installed binaries
func managedBinDir() (string, error) {
	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("APPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Roaming")
		}
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local")
	}
	return filepath.Join(base, "lurus-switch", "bin"), nil
}

// toolCacheDir returns the cache directory for a specific tool
func toolCacheDir(toolName string) string {
	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			home, _ := os.UserHomeDir()
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(localAppData, "lurus-switch", "cache", toolName)
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Caches", "lurus-switch", toolName)
	default:
		xdgCache := os.Getenv("XDG_CACHE_HOME")
		if xdgCache == "" {
			home, _ := os.UserHomeDir()
			xdgCache = filepath.Join(home, ".cache")
		}
		return filepath.Join(xdgCache, "lurus-switch", toolName)
	}
}

// addBinToPath attempts to add binDir to the Windows user PATH via setx (best-effort).
// setx does NOT expand %PATH%, so we must read the current value via PowerShell first.
func addBinToPath(binDir string) string {
	if runtime.GOOS != "windows" {
		return ""
	}

	// Read current user PATH from the registry (not the process env, which includes system PATH)
	readCmd := exec.Command("powershell", "-NoProfile", "-Command",
		`[Environment]::GetEnvironmentVariable('PATH', 'User')`)
	hideWindow(readCmd)
	out, err := readCmd.Output()
	if err != nil {
		return fmt.Sprintf("Note: Could not read current PATH. Please add %s to your PATH manually.", binDir)
	}

	currentPath := strings.TrimSpace(string(out))

	// Check if binDir is already in PATH (case-insensitive)
	for _, entry := range strings.Split(currentPath, ";") {
		if strings.EqualFold(strings.TrimSpace(entry), binDir) {
			return "" // already present
		}
	}

	newPath := currentPath
	if newPath != "" {
		newPath += ";"
	}
	newPath += binDir

	// setx has a 1024-char limit; if exceeded, give manual instructions
	if len(newPath) > 1024 {
		return fmt.Sprintf("Note: PATH too long for setx. Please add %s to your PATH manually.", binDir)
	}

	setCmd := exec.Command("setx", "PATH", newPath)
	hideWindow(setCmd)
	if err := setCmd.Run(); err != nil {
		return fmt.Sprintf("Note: Could not update PATH automatically. Please add %s to your PATH manually.", binDir)
	}
	return "PATH updated. Restart your terminal for the change to take effect."
}

// binaryFilename returns the platform-specific binary filename for a given base name
func binaryFilename(baseName string) string {
	if runtime.GOOS == "windows" {
		return baseName + ".exe"
	}
	return baseName
}

// verifySHA256 computes the SHA-256 hash of the file at filePath and compares it
// against expectedHex (lowercase hex-encoded). Returns nil if expectedHex is empty
// (no checksum provided) or if the hashes match.
func verifySHA256(filePath, expectedHex string) error {
	if expectedHex == "" {
		return nil
	}
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open for SHA256 check: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("read for SHA256 check: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expectedHex) {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHex, actual)
	}
	return nil
}

// downloadAndInstallBinary performs the full flow: fetch latest release, download, extract, install.
// overrideURL, if non-empty, skips the GitHub API call and downloads directly from that URL.
// expectedSHA256, if non-empty, is verified against the downloaded file before extraction.
// progressFn, if non-nil, is called with (downloaded, total, percent) after each 64 KB chunk.
func downloadAndInstallBinary(ctx context.Context, cfg BinaryToolConfig, overrideURL, expectedSHA256 string, progressFn func(int64, int64, int)) (*InstallResult, error) {
	var dlURL, assetName string

	if overrideURL != "" {
		dlURL = overrideURL
		// Derive a local filename from the URL tail.
		parts := strings.Split(overrideURL, "/")
		assetName = parts[len(parts)-1]
		if assetName == "" {
			assetName = cfg.Name + "-download"
		}
	} else {
		release, err := fetchLatestRelease(ctx, cfg.GitHubOwner, cfg.GitHubRepo)
		if err != nil {
			return &InstallResult{Tool: cfg.Name, Success: false, Message: err.Error()}, nil
		}
		dlURL, assetName = findPlatformAsset(release.Assets)
		if dlURL == "" {
			return &InstallResult{
				Tool:    cfg.Name,
				Success: false,
				Message: fmt.Sprintf("no compatible binary asset found in release %s for %s/%s", release.TagName, runtime.GOOS, runtime.GOARCH),
			}, nil
		}
	}

	// Download to cache
	cacheDir := toolCacheDir(cfg.Name)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return &InstallResult{Tool: cfg.Name, Success: false, Message: fmt.Sprintf("failed to create cache dir: %v", err)}, nil
	}

	cachedPath := filepath.Join(cacheDir, assetName)
	dlOpts := downloader.Options{ProgressFn: progressFn}
	if err := downloader.DownloadFile(ctx, dlURL, cachedPath, dlOpts); err != nil {
		return &InstallResult{Tool: cfg.Name, Success: false, Message: fmt.Sprintf("download failed: %v", err)}, nil
	}

	// Verify integrity before extraction
	if err := verifySHA256(cachedPath, expectedSHA256); err != nil {
		os.Remove(cachedPath)
		return &InstallResult{Tool: cfg.Name, Success: false, Message: fmt.Sprintf("integrity check failed: %v", err)}, nil
	}

	// Resolve destination
	binDir, err := managedBinDir()
	if err != nil {
		return &InstallResult{Tool: cfg.Name, Success: false, Message: fmt.Sprintf("failed to resolve bin dir: %v", err)}, nil
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return &InstallResult{Tool: cfg.Name, Success: false, Message: fmt.Sprintf("failed to create bin dir: %v", err)}, nil
	}

	binaryDest := filepath.Join(binDir, binaryFilename(cfg.BinaryName))

	// Extract or copy binary
	binFile := binaryFilename(cfg.BinaryName)
	lowerAsset := strings.ToLower(assetName)
	switch {
	case strings.HasSuffix(lowerAsset, ".zip"):
		if err := extractFromZip(cachedPath, binFile, binaryDest); err != nil {
			return &InstallResult{Tool: cfg.Name, Success: false, Message: fmt.Sprintf("extraction failed: %v", err)}, nil
		}
	case strings.HasSuffix(lowerAsset, ".tar.gz") || strings.HasSuffix(lowerAsset, ".tgz"):
		if err := extractFromTarGz(cachedPath, binFile, binaryDest); err != nil {
			return &InstallResult{Tool: cfg.Name, Success: false, Message: fmt.Sprintf("tar.gz extraction failed: %v", err)}, nil
		}
	default:
		// Assume the downloaded file is the binary itself (e.g. a standalone .exe)
		if err := copyBinaryFile(cachedPath, binaryDest); err != nil {
			return &InstallResult{Tool: cfg.Name, Success: false, Message: fmt.Sprintf("failed to copy binary: %v", err)}, nil
		}
	}

	// Best-effort PATH update
	pathMsg := addBinToPath(binDir)

	return &InstallResult{
		Tool:    cfg.Name,
		Success: true,
		Message: fmt.Sprintf("installed to %s. %s", binaryDest, pathMsg),
	}, nil
}

// findBinaryExecutable locates a binary by name, checking PATH then the managed bin dir
func findBinaryExecutable(name string) (string, error) {
	exeName := binaryFilename(name)

	if path, err := exec.LookPath(exeName); err == nil {
		return path, nil
	}

	// Check the managed bin directory
	if binDir, err := managedBinDir(); err == nil {
		candidate := filepath.Join(binDir, exeName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("%s executable not found", name)
}

// removeManagedBinary removes a binary from the managed bin directory
func removeManagedBinary(binaryName string) error {
	binDir, err := managedBinDir()
	if err != nil {
		return err
	}
	path := filepath.Join(binDir, binaryFilename(binaryName))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove binary: %w", err)
	}
	return nil
}
