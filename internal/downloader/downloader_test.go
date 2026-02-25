package downloader

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// === Constructor Tests ===

func TestNewDownloader(t *testing.T) {
	tmpDir := t.TempDir()
	d := NewDownloader(tmpDir)

	if d == nil {
		t.Fatal("NewDownloader should return non-nil downloader")
	}
	if d.cacheDir != tmpDir {
		t.Errorf("Expected cacheDir '%s', got '%s'", tmpDir, d.cacheDir)
	}
	if d.client == nil {
		t.Error("HTTP client should not be nil")
	}
}

// === GetCacheDir Tests ===

func TestGetCacheDir(t *testing.T) {
	// This test will use the actual system paths
	// Just verify it returns a non-empty string without error
	cacheDir, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir failed: %v", err)
	}

	if cacheDir == "" {
		t.Error("Cache directory should not be empty")
	}

	// Verify directory was created
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("Cache directory should be created")
	}
}

func TestGetCacheDir_ContainsLurusSwitch(t *testing.T) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir failed: %v", err)
	}

	if !containsPath(cacheDir, "lurus-switch") {
		t.Error("Cache directory should contain 'lurus-switch'")
	}
}

func TestGetCacheDir_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	cacheDir, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir failed: %v", err)
	}

	// On Windows, should use LOCALAPPDATA
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" && !hasPrefix(cacheDir, localAppData) {
		t.Error("Windows cache should be under LOCALAPPDATA")
	}
}

func TestGetCacheDir_Darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test")
	}

	cacheDir, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir failed: %v", err)
	}

	if !containsPath(cacheDir, "Library/Caches") {
		t.Error("macOS cache should be under ~/Library/Caches")
	}
}

func TestGetCacheDir_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}

	cacheDir, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir failed: %v", err)
	}

	xdgCache := os.Getenv("XDG_CACHE_HOME")
	if xdgCache != "" {
		if !hasPrefix(cacheDir, xdgCache) {
			t.Error("Linux cache should respect XDG_CACHE_HOME")
		}
	} else {
		if !containsPath(cacheDir, ".cache") {
			t.Error("Linux cache should be under ~/.cache")
		}
	}
}

// === Download Tests ===

func TestDownload_Success(t *testing.T) {
	// Create a test server
	testContent := "test file content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	d := NewDownloader(tmpDir)

	result, err := d.Download(server.URL+"/test.txt", "test.txt")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Size != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), result.Size)
	}

	// Verify file content
	content, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(content))
	}
}

func TestDownload_CacheHit(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-create a cached file
	cachedContent := "cached content"
	cachedPath := filepath.Join(tmpDir, "cached.txt")
	if err := os.WriteFile(cachedPath, []byte(cachedContent), 0644); err != nil {
		t.Fatalf("Failed to create cached file: %v", err)
	}

	d := NewDownloader(tmpDir)

	// Should return cached file without making HTTP request
	result, err := d.Download("http://should-not-be-called/cached.txt", "cached.txt")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if result.Path != cachedPath {
		t.Errorf("Expected cached path '%s', got '%s'", cachedPath, result.Path)
	}
}

func TestDownload_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	d := NewDownloader(tmpDir)

	_, err := d.Download(server.URL+"/notfound.txt", "notfound.txt")
	if err == nil {
		t.Error("Expected error for HTTP 404")
	}
	if !containsPath(err.Error(), "404") {
		t.Error("Error should mention HTTP status code")
	}
}

func TestDownload_NetworkFailure(t *testing.T) {
	tmpDir := t.TempDir()
	d := NewDownloader(tmpDir)

	// Use an invalid URL to trigger network failure
	_, err := d.Download("http://invalid.localhost.test:99999/file.txt", "file.txt")
	if err == nil {
		t.Error("Expected error for network failure")
	}
}

func TestDownload_CreatesNestedDirectories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	d := NewDownloader(tmpDir)

	// File in nested directory
	result, err := d.Download(server.URL+"/nested/dir/file.txt", "nested/dir/file.txt")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "nested", "dir", "file.txt")
	if result.Path != expectedPath {
		t.Errorf("Expected path '%s', got '%s'", expectedPath, result.Path)
	}
}

// === FetchJSON Tests ===

func TestFetchJSON_Success(t *testing.T) {
	expected := map[string]interface{}{
		"name":    "test",
		"version": "1.0.0",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	d := NewDownloader(tmpDir)

	var result map[string]interface{}
	err := d.FetchJSON(server.URL+"/api/data", &result)
	if err != nil {
		t.Fatalf("FetchJSON failed: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("Expected name 'test', got '%v'", result["name"])
	}
	if result["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%v'", result["version"])
	}
}

func TestFetchJSON_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	d := NewDownloader(tmpDir)

	var result map[string]interface{}
	err := d.FetchJSON(server.URL+"/api/error", &result)
	if err == nil {
		t.Error("Expected error for HTTP 500")
	}
}

func TestFetchJSON_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	d := NewDownloader(tmpDir)

	var result map[string]interface{}
	err := d.FetchJSON(server.URL+"/api/invalid", &result)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
	if !containsPath(err.Error(), "parse JSON") {
		t.Error("Error should mention JSON parsing")
	}
}

func TestFetchJSON_NetworkFailure(t *testing.T) {
	tmpDir := t.TempDir()
	d := NewDownloader(tmpDir)

	var result map[string]interface{}
	err := d.FetchJSON("http://invalid.localhost.test:99999/api", &result)
	if err == nil {
		t.Error("Expected error for network failure")
	}
}

// === ClearCache Tests ===

func TestClearCache_Success(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	os.MkdirAll(cacheDir, 0755)

	// Create some files in cache
	os.WriteFile(filepath.Join(cacheDir, "file1.txt"), []byte("content1"), 0644)
	os.MkdirAll(filepath.Join(cacheDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(cacheDir, "subdir", "file2.txt"), []byte("content2"), 0644)

	d := NewDownloader(cacheDir)

	err := d.ClearCache()
	if err != nil {
		t.Fatalf("ClearCache failed: %v", err)
	}

	// Verify cache directory is removed
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Error("Cache directory should be removed")
	}
}

func TestClearCache_DirNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "nonexistent")

	d := NewDownloader(nonExistentDir)

	// Should not error when directory doesn't exist
	err := d.ClearCache()
	if err != nil {
		t.Errorf("ClearCache should not error for non-existent directory: %v", err)
	}
}

// === DownloadResult Tests ===

func TestDownloadResult_Serialization(t *testing.T) {
	result := DownloadResult{
		Path:    "/path/to/file",
		Size:    12345,
		Version: "1.0.0",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded DownloadResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Path != result.Path {
		t.Error("Path mismatch")
	}
	if decoded.Size != result.Size {
		t.Error("Size mismatch")
	}
	if decoded.Version != result.Version {
		t.Error("Version mismatch")
	}
}

// === Helper Functions ===

func containsPath(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
