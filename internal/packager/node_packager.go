package packager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// NodePackager handles packaging Gemini CLI with Node.js pkg
type NodePackager struct {
	npmPath string
	npxPath string
}

// NewNodePackager creates a new Node packager
func NewNodePackager() (*NodePackager, error) {
	npmPath, err := findNpm()
	if err != nil {
		return nil, err
	}

	npxPath, err := findNpx()
	if err != nil {
		return nil, err
	}

	return &NodePackager{
		npmPath: npmPath,
		npxPath: npxPath,
	}, nil
}

// findNpm locates the npm executable
func findNpm() (string, error) {
	// Check PATH first
	if path, err := exec.LookPath("npm"); err == nil {
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
		programFiles := os.Getenv("ProgramFiles")
		candidates = []string{
			filepath.Join(programFiles, "nodejs", "npm.cmd"),
			filepath.Join(os.Getenv("APPDATA"), "npm", "npm.cmd"),
			filepath.Join(home, ".bun", "bin", "npm"),
		}
	default:
		candidates = []string{
			"/usr/local/bin/npm",
			"/usr/bin/npm",
			filepath.Join(home, ".nvm", "current", "bin", "npm"),
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("npm not found - please install Node.js")
}

// findNpx locates the npx executable
func findNpx() (string, error) {
	// Check PATH first
	if path, err := exec.LookPath("npx"); err == nil {
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
		programFiles := os.Getenv("ProgramFiles")
		candidates = []string{
			filepath.Join(programFiles, "nodejs", "npx.cmd"),
			filepath.Join(os.Getenv("APPDATA"), "npm", "npx.cmd"),
			filepath.Join(home, ".bun", "bin", "npx"),
		}
	default:
		candidates = []string{
			"/usr/local/bin/npx",
			"/usr/bin/npx",
			filepath.Join(home, ".nvm", "current", "bin", "npx"),
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("npx not found - please install Node.js")
}

// Package creates a standalone executable for Gemini CLI wrapper
func (p *NodePackager) Package(configDir, outputPath string) error {
	// Create a temporary directory for the wrapper project
	tmpDir, err := os.MkdirTemp("", "gemini-package-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the wrapper script
	wrapperPath := filepath.Join(tmpDir, "gemini-wrapper.js")
	wrapperContent := p.generateWrapper(configDir)
	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0644); err != nil {
		return fmt.Errorf("failed to write wrapper script: %w", err)
	}

	// Create package.json
	packageJSON := fmt.Sprintf(`{
  "name": "gemini-cli-custom",
  "version": "1.0.0",
  "bin": "gemini-wrapper.js",
  "pkg": {
    "targets": ["%s"],
    "outputPath": "%s"
  },
  "dependencies": {}
}
`, p.getPkgTarget(), filepath.Base(outputPath))

	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		return fmt.Errorf("failed to write package.json: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Package with pkg using npx
	pkgCmd := exec.Command(p.npxPath, "pkg", ".", "--output", outputPath)
	pkgCmd.Dir = tmpDir
	pkgCmd.Stdout = os.Stdout
	pkgCmd.Stderr = os.Stderr

	if err := pkgCmd.Run(); err != nil {
		return fmt.Errorf("failed to package with pkg: %w", err)
	}

	return nil
}

// generateWrapper creates the JavaScript wrapper script content
func (p *NodePackager) generateWrapper(configDir string) string {
	// Escape backslashes for Windows paths
	escapedPath := configDir
	if runtime.GOOS == "windows" {
		escapedPath = ""
		for _, c := range configDir {
			if c == '\\' {
				escapedPath += "\\\\"
			} else {
				escapedPath += string(c)
			}
		}
	}

	return fmt.Sprintf(`#!/usr/bin/env node
const { spawn } = require("child_process");
const fs = require("fs");
const path = require("path");

const CONFIG_DIR = "%s";

async function main() {
  // Check if GEMINI.md exists
  const geminiMdPath = path.join(CONFIG_DIR, "GEMINI.md");
  const settingsPath = path.join(CONFIG_DIR, "gemini-settings.json");

  if (!fs.existsSync(geminiMdPath) && !fs.existsSync(settingsPath)) {
    console.error("Error: Configuration files not found in", CONFIG_DIR);
    process.exit(1);
  }

  // Read settings if available
  let settings = {};
  if (fs.existsSync(settingsPath)) {
    settings = JSON.parse(fs.readFileSync(settingsPath, "utf-8"));
  }

  // Build command arguments
  const args = process.argv.slice(2);

  // Add model if specified
  if (settings.model && !args.includes("--model")) {
    args.unshift("--model", settings.model);
  }

  // Set up environment
  const env = { ...process.env };

  // Add config directory to environment
  env.GEMINI_CONFIG_DIR = CONFIG_DIR;

  // Launch Gemini CLI
  const gemini = spawn("gemini", args, {
    stdio: "inherit",
    env: env,
  });

  gemini.on("error", (err) => {
    if (err.code === "ENOENT") {
      console.error("Error: Gemini CLI not found. Please install it first.");
      console.error("  npm install -g @google/gemini-cli");
    } else {
      console.error("Error:", err.message);
    }
    process.exit(1);
  });

  gemini.on("exit", (code) => {
    process.exit(code ?? 0);
  });
}

main().catch((err) => {
  console.error("Error:", err.message);
  process.exit(1);
});
`, escapedPath)
}

// getPkgTarget returns the pkg target for the current platform
func (p *NodePackager) getPkgTarget() string {
	var os, arch string

	switch runtime.GOOS {
	case "windows":
		os = "win"
	case "darwin":
		os = "macos"
	case "linux":
		os = "linux"
	default:
		os = "linux"
	}

	switch runtime.GOARCH {
	case "amd64":
		arch = "x64"
	case "arm64":
		arch = "arm64"
	default:
		arch = "x64"
	}

	return fmt.Sprintf("node18-%s-%s", os, arch)
}

// GetNpmPath returns the path to npm
func (p *NodePackager) GetNpmPath() string {
	return p.npmPath
}

// GetNpxPath returns the path to npx
func (p *NodePackager) GetNpxPath() string {
	return p.npxPath
}

// IsNodeInstalled checks if Node.js is available
func IsNodeInstalled() bool {
	_, err := findNpm()
	return err == nil
}
