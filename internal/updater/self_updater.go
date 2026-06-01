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

// maxDownloadBytes caps the binary download to 500 MB to prevent unbounded reads.
// Exposed as a var so tests can override it without unsafe casting.
var maxDownloadBytes int64 = 500 * 1024 * 1024

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

// applyWindows uses a batch script to replace the locked executable and restart.
//
// Bat script order: (1) wait, (2) copy new → current (keeps current intact if copy fails),
// (3) rename current to .bak as a fallback copy, then launch. On launch the running
// new binary can clean up .bak on next start. This avoids the del-before-move window
// where a crash leaves no binary at all.
func (s *SelfUpdater) applyWindows(currentExe, newExe string) error {
	bakPath := currentExe + ".bak"
	batPath := currentExe + ".update.bat"

	// Batch script:
	//   1. Wait for the current process to exit.
	//   2. Rename current exe → .bak (preserves it in case the new binary fails to launch).
	//   3. Move new exe → current exe path.
	//   4. Launch the new exe.
	//   5. Self-delete the bat.
	//
	// If step 4 fails the user still has .bak to restore manually.
	batContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
move /y "%s" "%s"
move /y "%s" "%s"
start "" "%s"
del "%%~f0"
`, currentExe, bakPath, newExe, currentExe, currentExe)

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

// applyUnix directly replaces the binary and restarts.
// Before overwriting, the current executable is backed up to currentExe+".bak".
// If the new binary fails to start the caller should restore from the .bak.
func (s *SelfUpdater) applyUnix(currentExe, newExe string) error {
	bakPath := currentExe + ".bak"

	if err := os.Chmod(newExe, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	// Preserve the current binary as a rollback copy before overwriting.
	if err := os.Rename(currentExe, bakPath); err != nil {
		return fmt.Errorf("failed to back up current executable: %w", err)
	}

	if err := os.Rename(newExe, currentExe); err != nil {
		// Attempt to restore the backup before returning the error.
		_ = os.Rename(bakPath, currentExe)
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	// Restart the application
	cmd := exec.Command(currentExe)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		// Restore the backup so the user is not left with a broken installation.
		_ = os.Rename(bakPath, currentExe)
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

	if _, err := io.Copy(out, io.LimitReader(resp.Body, maxDownloadBytes)); err != nil {
		return fmt.Errorf("failed to write downloaded data: %w", err)
	}

	return nil
}

// GetCurrentVersion returns the current app version
func (s *SelfUpdater) GetCurrentVersion() string {
	return s.currentVersion
}
