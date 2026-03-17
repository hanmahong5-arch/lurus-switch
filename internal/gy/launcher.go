package gy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"lurus-switch/internal/downloader"
	"lurus-switch/internal/toolmanifest"
)

// ErrNotInstalled is returned when a desktop product exe cannot be found.
var ErrNotInstalled = errors.New("product not installed")

// Launch opens the product. For web/service products it opens the URL in the
// default browser. For desktop products it starts the local executable.
// The openBrowser function should be wails.BrowserOpenURL or equivalent.
func Launch(product GYProduct, openBrowser func(string), userToken string) error {
	switch product.Kind {
	case KindWeb:
		url := product.LaunchURL
		if userToken != "" {
			url += "?token=" + userToken
		}
		openBrowser(url)
		return nil

	case KindService:
		openBrowser(product.ServiceURL)
		return nil

	case KindDesktop:
		exePath, err := FindCreatorExe()
		if err != nil {
			return ErrNotInstalled
		}
		cmd := exec.Command(exePath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Start()

	default:
		return fmt.Errorf("unknown product kind: %s", product.Kind)
	}
}

// FindCreatorExe searches common installation paths for lurus-creator.exe (Windows)
// or lurus-creator (Unix).
func FindCreatorExe() (string, error) {
	exeName := "lurus-creator"
	if runtime.GOOS == "windows" {
		exeName = "lurus-creator.exe"
	}

	// Search candidates in priority order
	candidates := []string{}

	// 1. Same directory as current executable
	if self, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(self), exeName))
	}

	// 2. Standard install locations
	switch runtime.GOOS {
	case "windows":
		programFiles := os.Getenv("ProgramFiles")
		if programFiles == "" {
			programFiles = `C:\Program Files`
		}
		candidates = append(candidates,
			filepath.Join(programFiles, "Lurus Creator", exeName),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "lurus-creator", exeName),
		)
	case "darwin":
		candidates = append(candidates,
			"/Applications/Lurus Creator.app/Contents/MacOS/lurus-creator",
		)
	default:
		home, _ := os.UserHomeDir()
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", exeName),
			"/usr/local/bin/"+exeName,
		)
	}

	// 3. PATH lookup
	if path, err := exec.LookPath(exeName); err == nil {
		candidates = append(candidates, path)
	}

	for _, p := range candidates {
		if p == "" {
			continue
		}
		info, err := os.Stat(p)
		if err == nil && info.Mode().IsRegular() {
			return p, nil
		}
	}

	return "", ErrNotInstalled
}

// DownloadCreator downloads the Lurus Creator installer for the current platform
// using the provided manifest, saves it to destDir, then launches the installer.
// progressFn is called with 0-100 percent values during the download; may be nil.
func DownloadCreator(ctx context.Context, mf *toolmanifest.Manifest, destDir string, progressFn func(pct int)) error {
	if mf == nil {
		return fmt.Errorf("tool manifest not available — please wait for app startup to complete and retry")
	}

	entry, ok := mf.Tools["lurus-creator"]
	if !ok {
		return fmt.Errorf("lurus-creator not found in tool manifest")
	}

	platform := toolmanifest.CurrentPlatform()
	asset, ok := entry.Platforms[platform]
	if !ok {
		return fmt.Errorf("lurus-creator is not available for the current platform (%s)", platform)
	}

	// Derive filename from URL tail
	parts := strings.Split(asset.URL, "/")
	filename := parts[len(parts)-1]
	if filename == "" {
		filename = "lurus-creator-installer"
	}
	destPath := filepath.Join(destDir, filename)

	opts := downloader.Options{
		ProgressFn: func(_, _ int64, pct int) {
			if progressFn != nil {
				progressFn(pct)
			}
		},
	}
	if err := downloader.DownloadFile(ctx, asset.URL, destPath, opts); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	return launchInstaller(destPath)
}

// launchInstaller runs a downloaded installer file in the background.
// Windows: executes the .exe installer directly.
// macOS: uses `open` to mount and present the .dmg.
func launchInstaller(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		// Windows and Linux: run the installer directly
		cmd = exec.Command(path)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
