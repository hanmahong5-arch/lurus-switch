package updater

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const (
	// GitHub repo for the Switch app itself
	selfUpdateOwner = "lurus-dev"
	selfUpdateRepo  = "lurus-switch"

	downloadTimeout = 120 * time.Second
)

// SelfUpdater handles checking and applying updates for the Switch app
type SelfUpdater struct {
	checker        *GitHubChecker
	currentVersion string
}

// NewSelfUpdater creates a new SelfUpdater with the given app version
func NewSelfUpdater(appVersion string) *SelfUpdater {
	return &SelfUpdater{
		checker:        NewGitHubChecker(selfUpdateOwner, selfUpdateRepo),
		currentVersion: appVersion,
	}
}

// CheckUpdate checks if a newer version of Switch is available on GitHub
func (s *SelfUpdater) CheckUpdate() (*UpdateInfo, error) {
	return s.checker.CheckUpdate("lurus-switch", s.currentVersion)
}

// ApplyUpdate downloads and replaces the current executable.
// On Windows, it uses a .bat script for delayed replacement since the running .exe is locked.
func (s *SelfUpdater) ApplyUpdate() error {
	info, err := s.CheckUpdate()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	if !info.UpdateAvailable {
		return fmt.Errorf("no update available (current: %s, latest: %s)", info.CurrentVersion, info.LatestVersion)
	}
	if info.DownloadURL == "" {
		return fmt.Errorf("no download URL found for the latest release")
	}

	// Download new binary to temp file
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	tmpPath := currentExe + ".new"
	if err := downloadFile(info.DownloadURL, tmpPath); err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	// Verify integrity before applying.
	verifyClient := &http.Client{Timeout: downloadTimeout}
	if err := VerifyFileChecksum(verifyClient, info.DownloadURL, tmpPath); err != nil {
		return fmt.Errorf("update integrity check failed: %w", err)
	}

	if runtime.GOOS == "windows" {
		return s.applyWindows(currentExe, tmpPath)
	}
	return s.applyUnix(currentExe, tmpPath)
}

// applyWindows uses a batch script to replace the locked executable and restart
func (s *SelfUpdater) applyWindows(currentExe, newExe string) error {
	batPath := currentExe + ".update.bat"
	batContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
del "%s"
move "%s" "%s"
start "" "%s"
del "%%~f0"
`, currentExe, newExe, currentExe, currentExe)

	if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
		return fmt.Errorf("failed to write update script: %w", err)
	}

	cmd := exec.Command("cmd", "/c", "start", "/b", batPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch update script: %w", err)
	}

	// Exit the current process so the bat script can replace the exe
	os.Exit(0)
	return nil
}

// applyUnix directly replaces the binary and restarts
func (s *SelfUpdater) applyUnix(currentExe, newExe string) error {
	if err := os.Chmod(newExe, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	if err := os.Rename(newExe, currentExe); err != nil {
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	// Restart the application
	cmd := exec.Command(currentExe)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to restart application: %w", err)
	}

	os.Exit(0)
	return nil
}

// downloadFile downloads a URL to a local file
func downloadFile(url, destPath string) error {
	client := &http.Client{Timeout: downloadTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP GET failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write downloaded data: %w", err)
	}

	return nil
}

// GetCurrentVersion returns the current app version
func (s *SelfUpdater) GetCurrentVersion() string {
	return s.currentVersion
}
