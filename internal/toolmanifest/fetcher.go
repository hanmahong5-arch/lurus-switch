package toolmanifest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	manifestEndpoint   = "/api/v2/tools/download-manifest"
	defaultManifestAPI = "https://api.lurus.cn"
	cacheFilename      = "tool_manifest_cache.json"
	cacheTTL           = 6 * time.Hour
	fetchTimeout       = 5 * time.Second
)

// cacheEntry wraps a Manifest with a fetch timestamp for TTL checks.
type cacheEntry struct {
	FetchedAt time.Time `json:"fetched_at"`
	Manifest  Manifest  `json:"manifest"`
}

// Fetch retrieves the manifest using this priority order:
//  1. Valid local cache (age < cacheTTL)
//  2. Live HTTP fetch from apiBase + manifestEndpoint (falls back to api.lurus.cn if apiBase is empty)
//  3. Stale local cache (any age) as offline fallback
//  4. Compile-time Builtin()
//
// cacheDir is typically the app data base directory.
func Fetch(ctx context.Context, apiBase, cacheDir string) (*Manifest, error) {
	cachePath := filepath.Join(cacheDir, cacheFilename)

	// 1. Fresh cache
	if entry, err := readCache(cachePath); err == nil {
		if time.Since(entry.FetchedAt) < cacheTTL {
			m := entry.Manifest
			return &m, nil
		}
	}

	// 2. Live HTTP fetch (fall back to public API if apiBase is empty)
	fetchBase := apiBase
	if fetchBase == "" {
		fetchBase = defaultManifestAPI
	}
	if mf, err := fetchHTTP(ctx, fetchBase); err == nil {
		// Persist to cache (best-effort)
		_ = writeCache(cachePath, mf)
		return mf, nil
	}

	// 3. Stale cache as offline fallback
	if entry, err := readCache(cachePath); err == nil {
		m := entry.Manifest
		return &m, nil
	}

	// 4. Compile-time builtin
	return Builtin(), nil
}

// fetchHTTP performs the HTTP GET and decodes the manifest JSON.
func fetchHTTP(ctx context.Context, apiBase string) (*Manifest, error) {
	url := apiBase + manifestEndpoint
	reqCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "lurus-switch/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP %d from %s", resp.StatusCode, url)
	}

	var mf Manifest
	if err := json.NewDecoder(resp.Body).Decode(&mf); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	return &mf, nil
}

// readCache loads and deserialises the on-disk cache entry.
func readCache(path string) (*cacheEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("corrupt cache: %w", err)
	}
	return &entry, nil
}

// writeCache serialises the manifest and writes it to disk.
func writeCache(path string, mf *Manifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	entry := cacheEntry{FetchedAt: time.Now().UTC(), Manifest: *mf}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
