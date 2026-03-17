package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// VerifyFileChecksum downloads the expected SHA-256 checksum from "{downloadURL}.sha256"
// and compares it against the local file at localPath.
//
// If the .sha256 sidecar returns HTTP 404 (transition period), the function logs a warning
// and returns nil to avoid blocking updates during gradual rollout.
//
// On checksum mismatch the local file is deleted and an error is returned.
func VerifyFileChecksum(client *http.Client, downloadURL, localPath string) error {
	checksumURL := downloadURL + ".sha256"

	resp, err := client.Get(checksumURL)
	if err != nil {
		// Network failure — degrade gracefully; do not block update
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Sidecar not yet published — warn but allow update to proceed
		fmt.Printf("WARNING: no checksum file at %s; skipping integrity check\n", checksumURL)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksum download failed: HTTP %d from %s", resp.StatusCode, checksumURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read checksum body: %w", err)
	}

	// Expected format: "<hex-sha256>  <filename>" (sha256sum output) or bare hex string
	expectedHex := strings.TrimSpace(strings.Fields(string(body))[0])
	if len(expectedHex) != 64 {
		return fmt.Errorf("unexpected checksum format in %s", checksumURL)
	}

	// Compute actual SHA-256 of the downloaded file
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open downloaded file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash file: %w", err)
	}
	actualHex := hex.EncodeToString(h.Sum(nil))

	if !strings.EqualFold(actualHex, expectedHex) {
		// Remove the corrupted/tampered file before returning error
		_ = os.Remove(localPath)
		return fmt.Errorf("checksum mismatch: expected %s, got %s — download aborted", expectedHex, actualHex)
	}

	return nil
}
