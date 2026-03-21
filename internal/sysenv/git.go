package sysenv

import (
	"fmt"
	"os/exec"
	"strings"
)

// DetectGit checks whether git is installed and retrieves global configuration.
func DetectGit() (*GitInfo, error) {
	info := &GitInfo{}

	_, err := exec.LookPath("git")
	if err != nil {
		// git not found on PATH — not an error, just not installed.
		return info, nil
	}
	info.Installed = true

	// git --version
	if out, err := exec.Command("git", "--version").Output(); err == nil {
		info.Version = strings.TrimSpace(string(out))
	}

	// git config --global user.name
	if out, err := exec.Command("git", "config", "--global", "user.name").Output(); err == nil {
		info.UserName = strings.TrimSpace(string(out))
	}

	// git config --global user.email
	if out, err := exec.Command("git", "config", "--global", "user.email").Output(); err == nil {
		info.UserEmail = strings.TrimSpace(string(out))
	}

	return info, nil
}

// SetGitConfig sets a global git config key to the given value.
func SetGitConfig(key, value string) error {
	if key == "" {
		return fmt.Errorf("git config key must not be empty")
	}
	cmd := exec.Command("git", "config", "--global", key, value)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config --global %s %s failed: %s: %w", key, value, strings.TrimSpace(string(out)), err)
	}
	return nil
}
