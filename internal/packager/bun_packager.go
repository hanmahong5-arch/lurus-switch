package packager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BunPackager handles packaging Claude Code configurations with Bun
type BunPackager struct {
	bunPath string
}

// NewBunPackager creates a new Bun packager
func NewBunPackager() (*BunPackager, error) {
	bunPath, err := findBun()
	if err != nil {
		return nil, err
	}
	return &BunPackager{bunPath: bunPath}, nil
}

// findBun locates the Bun executable
func findBun() (string, error) {
	// Check PATH first
	if path, err := exec.LookPath("bun"); err == nil {
		return path, nil
	}

	// Check common installation locations
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	var candidates []string
	switch runtime.GOOS {
	case "windows":
		candidates = []string{
			filepath.Join(home, ".bun", "bin", "bun.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "bun", "bin", "bun.exe"),
		}
	default:
		candidates = []string{
			filepath.Join(home, ".bun", "bin", "bun"),
			"/usr/local/bin/bun",
			"/usr/bin/bun",
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("bun not found - please install Bun (https://bun.sh)")
}

// Package creates a standalone executable from a Claude Code wrapper script
func (p *BunPackager) Package(configDir, outputPath string) error {
	// Create a temporary directory for the wrapper project
	tmpDir, err := os.MkdirTemp("", "claude-package-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the wrapper script
	wrapperPath := filepath.Join(tmpDir, "claude-wrapper.ts")
	wrapperContent := p.generateWrapper(configDir)
	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0644); err != nil {
		return fmt.Errorf("failed to write wrapper script: %w", err)
	}

	// Create package.json
	packageJSON := `{
  "name": "claude-code-custom",
  "version": "1.0.0",
  "type": "module",
  "dependencies": {}
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		return fmt.Errorf("failed to write package.json: %w", err)
	}

	// Compile with Bun
	var compileCmd *exec.Cmd
	target := getCompileTarget()

	if target != "" {
		compileCmd = exec.Command(p.bunPath, "build", "--compile", "--target", target, wrapperPath, "--outfile", outputPath)
	} else {
		compileCmd = exec.Command(p.bunPath, "build", "--compile", wrapperPath, "--outfile", outputPath)
	}

	compileCmd.Dir = tmpDir
	compileCmd.Stdout = os.Stdout
	compileCmd.Stderr = os.Stderr

	if err := compileCmd.Run(); err != nil {
		return fmt.Errorf("failed to compile with Bun: %w", err)
	}

	return nil
}

// generateWrapper creates the TypeScript wrapper script content
func (p *BunPackager) generateWrapper(configDir string) string {
	// Escape backslashes for Windows paths
	escapedPath := strings.ReplaceAll(configDir, "\\", "\\\\")

	return fmt.Sprintf(`#!/usr/bin/env bun
import { spawn } from "child_process";
import { existsSync, readFileSync } from "fs";
import { join } from "path";

const CONFIG_DIR = "%s";

async function main() {
  // Check if config exists
  const settingsPath = join(CONFIG_DIR, "settings.json");
  if (!existsSync(settingsPath)) {
    console.error("Error: Configuration file not found at", settingsPath);
    process.exit(1);
  }

  // Read configuration
  const settings = JSON.parse(readFileSync(settingsPath, "utf-8"));

  // Build command arguments
  const args = process.argv.slice(2);

  // Add model if specified
  if (settings.model && !args.includes("--model")) {
    args.unshift("--model", settings.model);
  }

  // Launch Claude CLI
  const claude = spawn("claude", args, {
    stdio: "inherit",
    env: {
      ...process.env,
      CLAUDE_CONFIG_DIR: CONFIG_DIR,
      ...(settings.apiKey && { ANTHROPIC_API_KEY: settings.apiKey }),
    },
  });

  claude.on("exit", (code) => {
    process.exit(code ?? 0);
  });
}

main().catch((err) => {
  console.error("Error:", err.message);
  process.exit(1);
});
`, escapedPath)
}

// getCompileTarget returns the Bun compile target for the current platform
func getCompileTarget() string {
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "amd64" {
			return "bun-windows-x64"
		}
		return ""
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "bun-darwin-arm64"
		}
		return "bun-darwin-x64"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "bun-linux-arm64"
		}
		return "bun-linux-x64"
	default:
		return ""
	}
}

// GetBunPath returns the path to the Bun executable
func (p *BunPackager) GetBunPath() string {
	return p.bunPath
}

// IsBunInstalled checks if Bun is available
func IsBunInstalled() bool {
	_, err := findBun()
	return err == nil
}
