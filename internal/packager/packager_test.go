package packager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ============================================================================
// Bun Packager Tests
// ============================================================================

func TestIsBunInstalled(t *testing.T) {
	// Just test that the function doesn't panic
	// Result depends on whether Bun is installed on the system
	_ = IsBunInstalled()
}

func TestBunPackager_GenerateWrapper(t *testing.T) {
	p := &BunPackager{bunPath: "/usr/local/bin/bun"}
	configDir := "/path/to/config"

	wrapper := p.generateWrapper(configDir)

	// Verify wrapper contains expected content
	if !strings.Contains(wrapper, "#!/usr/bin/env bun") {
		t.Error("Wrapper should have bun shebang")
	}
	if !strings.Contains(wrapper, "/path/to/config") {
		t.Error("Wrapper should contain config dir")
	}
	if !strings.Contains(wrapper, "settings.json") {
		t.Error("Wrapper should reference settings.json")
	}
	if !strings.Contains(wrapper, "claude") {
		t.Error("Wrapper should spawn claude command")
	}
}

func TestBunPackager_GenerateWrapper_WindowsPath(t *testing.T) {
	p := &BunPackager{bunPath: "C:\\bun\\bun.exe"}
	configDir := "C:\\Users\\Test\\config"

	wrapper := p.generateWrapper(configDir)

	// Windows paths should be escaped
	if !strings.Contains(wrapper, "C:\\\\Users\\\\Test\\\\config") {
		t.Error("Windows paths should be escaped in wrapper")
	}
}

func TestBunPackager_GetBunPath(t *testing.T) {
	p := &BunPackager{bunPath: "/test/path/bun"}

	if p.GetBunPath() != "/test/path/bun" {
		t.Error("GetBunPath should return correct path")
	}
}

func TestGetCompileTarget_Windows_AMD64(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Skipping Windows AMD64 test")
	}

	target := getCompileTarget()
	if target != "bun-windows-x64" {
		t.Errorf("Expected 'bun-windows-x64', got '%s'", target)
	}
}

func TestGetCompileTarget_Darwin_ARM64(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("Skipping macOS ARM64 test")
	}

	target := getCompileTarget()
	if target != "bun-darwin-arm64" {
		t.Errorf("Expected 'bun-darwin-arm64', got '%s'", target)
	}
}

func TestGetCompileTarget_Darwin_AMD64(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "amd64" {
		t.Skip("Skipping macOS AMD64 test")
	}

	target := getCompileTarget()
	if target != "bun-darwin-x64" {
		t.Errorf("Expected 'bun-darwin-x64', got '%s'", target)
	}
}

func TestGetCompileTarget_Linux_AMD64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Skipping Linux AMD64 test")
	}

	target := getCompileTarget()
	if target != "bun-linux-x64" {
		t.Errorf("Expected 'bun-linux-x64', got '%s'", target)
	}
}

func TestGetCompileTarget_Linux_ARM64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		t.Skip("Skipping Linux ARM64 test")
	}

	target := getCompileTarget()
	if target != "bun-linux-arm64" {
		t.Errorf("Expected 'bun-linux-arm64', got '%s'", target)
	}
}

// ============================================================================
// Node Packager Tests
// ============================================================================

func TestIsNodeInstalled(t *testing.T) {
	// Just test that the function doesn't panic
	_ = IsNodeInstalled()
}

func TestNodePackager_GenerateWrapper(t *testing.T) {
	p := &NodePackager{bunPath: "/usr/local/bin/bun", npxPath: "/usr/local/bin/npx"}
	configDir := "/path/to/config"

	wrapper := p.generateWrapper(configDir)

	// Verify wrapper contains expected content
	if !strings.Contains(wrapper, "#!/usr/bin/env node") {
		t.Error("Wrapper should have node shebang")
	}
	if !strings.Contains(wrapper, configDir) {
		t.Error("Wrapper should contain config dir")
	}
	if !strings.Contains(wrapper, "GEMINI.md") {
		t.Error("Wrapper should reference GEMINI.md")
	}
	if !strings.Contains(wrapper, "gemini-settings.json") {
		t.Error("Wrapper should reference gemini-settings.json")
	}
	if !strings.Contains(wrapper, "gemini") {
		t.Error("Wrapper should spawn gemini command")
	}
}

func TestNodePackager_GenerateWrapper_WindowsPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	p := &NodePackager{bunPath: "bun", npxPath: "npx"}
	configDir := "C:\\Users\\Test\\config"

	wrapper := p.generateWrapper(configDir)

	// Windows paths should be escaped
	if !strings.Contains(wrapper, "C:\\\\Users\\\\Test\\\\config") {
		t.Error("Windows paths should be escaped in wrapper")
	}
}

func TestNodePackager_GetBunPath(t *testing.T) {
	p := &NodePackager{bunPath: "/test/bun", npxPath: "/test/npx"}

	if p.GetBunPath() != "/test/bun" {
		t.Error("GetBunPath should return correct path")
	}
}

func TestNodePackager_GetNpxPath(t *testing.T) {
	p := &NodePackager{bunPath: "/test/bun", npxPath: "/test/npx"}

	if p.GetNpxPath() != "/test/npx" {
		t.Error("GetNpxPath should return correct path")
	}
}

func TestNodePackager_GetPkgTarget(t *testing.T) {
	p := &NodePackager{}
	target := p.getPkgTarget()

	// Should contain node18
	if !strings.Contains(target, "node18") {
		t.Error("Pkg target should contain node18")
	}

	// Should contain OS
	switch runtime.GOOS {
	case "windows":
		if !strings.Contains(target, "win") {
			t.Error("Windows target should contain 'win'")
		}
	case "darwin":
		if !strings.Contains(target, "macos") {
			t.Error("macOS target should contain 'macos'")
		}
	case "linux":
		if !strings.Contains(target, "linux") {
			t.Error("Linux target should contain 'linux'")
		}
	}

	// Should contain arch
	switch runtime.GOARCH {
	case "amd64":
		if !strings.Contains(target, "x64") {
			t.Error("AMD64 target should contain 'x64'")
		}
	case "arm64":
		if !strings.Contains(target, "arm64") {
			t.Error("ARM64 target should contain 'arm64'")
		}
	}
}

// ============================================================================
// Rust Packager Tests
// ============================================================================

func TestNewRustPackager(t *testing.T) {
	p, err := NewRustPackager()
	if err != nil {
		// This might fail if cache directory can't be created
		// which is acceptable in some test environments
		t.Skipf("Skipping: %v", err)
	}

	if p == nil {
		t.Fatal("NewRustPackager should return non-nil packager")
	}

	if p.cacheDir == "" {
		t.Error("Cache directory should not be empty")
	}
}

func TestRustPackager_GetCacheDir(t *testing.T) {
	tmpDir := t.TempDir()
	p := &RustPackager{cacheDir: tmpDir}

	if p.GetCacheDir() != tmpDir {
		t.Error("GetCacheDir should return correct path")
	}
}

func TestRustPackager_GetAssetName(t *testing.T) {
	p := &RustPackager{}
	name := p.getAssetName()

	// Result depends on current platform
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "amd64" {
			if name != "codex-windows-x64.exe" {
				t.Errorf("Expected 'codex-windows-x64.exe', got '%s'", name)
			}
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			if name != "codex-darwin-arm64" {
				t.Errorf("Expected 'codex-darwin-arm64', got '%s'", name)
			}
		} else if runtime.GOARCH == "amd64" {
			if name != "codex-darwin-x64" {
				t.Errorf("Expected 'codex-darwin-x64', got '%s'", name)
			}
		}
	case "linux":
		if runtime.GOARCH == "arm64" {
			if name != "codex-linux-arm64" {
				t.Errorf("Expected 'codex-linux-arm64', got '%s'", name)
			}
		} else if runtime.GOARCH == "amd64" {
			if name != "codex-linux-x64" {
				t.Errorf("Expected 'codex-linux-x64', got '%s'", name)
			}
		}
	}
}

func TestRustPackager_MatchesAsset_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	p := &RustPackager{}

	testCases := []struct {
		name     string
		expected bool
	}{
		{"codex-windows-x64.exe", runtime.GOARCH == "amd64"},
		{"codex-win-x64.zip", runtime.GOARCH == "amd64"},
		{"codex-linux-x64", false},
		{"codex-darwin-arm64", false},
	}

	for _, tc := range testCases {
		result := p.matchesAsset(tc.name)
		if result != tc.expected {
			t.Errorf("matchesAsset(%s) = %v, expected %v", tc.name, result, tc.expected)
		}
	}
}

func TestRustPackager_MatchesAsset_Darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test")
	}

	p := &RustPackager{}

	testCases := []struct {
		name     string
		expected bool
	}{
		{"codex-darwin-arm64", runtime.GOARCH == "arm64"},
		{"codex-darwin-x64", runtime.GOARCH == "amd64"},
		{"codex-macos-arm64", runtime.GOARCH == "arm64"},
		{"codex-apple-aarch64", runtime.GOARCH == "arm64"},
		{"codex-linux-x64", false},
		{"codex-windows-x64.exe", false},
	}

	for _, tc := range testCases {
		result := p.matchesAsset(tc.name)
		if result != tc.expected {
			t.Errorf("matchesAsset(%s) = %v, expected %v", tc.name, result, tc.expected)
		}
	}
}

func TestRustPackager_MatchesAsset_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}

	p := &RustPackager{}

	testCases := []struct {
		name     string
		expected bool
	}{
		{"codex-linux-x64", runtime.GOARCH == "amd64"},
		{"codex-linux-amd64", runtime.GOARCH == "amd64"},
		{"codex-linux-x86_64", runtime.GOARCH == "amd64"},
		{"codex-linux-arm64", runtime.GOARCH == "arm64"},
		{"codex-linux-aarch64", runtime.GOARCH == "arm64"},
		{"codex-darwin-arm64", false},
		{"codex-windows-x64.exe", false},
	}

	for _, tc := range testCases {
		result := p.matchesAsset(tc.name)
		if result != tc.expected {
			t.Errorf("matchesAsset(%s) = %v, expected %v", tc.name, result, tc.expected)
		}
	}
}

func TestRustPackager_DownloadFile_Success(t *testing.T) {
	testContent := "binary content here"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	p := &RustPackager{cacheDir: tmpDir}

	destPath := filepath.Join(tmpDir, "downloaded.bin")
	err := p.downloadFile(server.URL+"/file.bin", destPath)
	if err != nil {
		t.Fatalf("downloadFile failed: %v", err)
	}

	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(content))
	}
}

func TestRustPackager_DownloadFile_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	p := &RustPackager{cacheDir: tmpDir}

	destPath := filepath.Join(tmpDir, "notfound.bin")
	err := p.downloadFile(server.URL+"/notfound", destPath)
	if err == nil {
		t.Error("Expected error for HTTP 404")
	}
}

func TestRustPackager_CopyFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	p := &RustPackager{cacheDir: tmpDir}

	// Create source file
	srcContent := "source file content"
	srcPath := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(srcPath, []byte(srcContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	dstPath := filepath.Join(tmpDir, "dest.txt")
	err := p.copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(content) != srcContent {
		t.Errorf("Expected content '%s', got '%s'", srcContent, string(content))
	}
}

func TestRustPackager_CopyFile_SourceNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	p := &RustPackager{cacheDir: tmpDir}

	err := p.copyFile(filepath.Join(tmpDir, "nonexistent"), filepath.Join(tmpDir, "dest"))
	if err == nil {
		t.Error("Expected error for non-existent source")
	}
}

func TestRustPackager_CopyFile_PreservesPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	p := &RustPackager{cacheDir: tmpDir}

	// Create source file with executable permission
	srcPath := filepath.Join(tmpDir, "source.sh")
	if err := os.WriteFile(srcPath, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	dstPath := filepath.Join(tmpDir, "dest.sh")
	err := p.copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Check permissions
	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("Failed to stat destination file: %v", err)
	}

	if info.Mode()&0100 == 0 {
		t.Error("Executable permission should be preserved")
	}
}

// ============================================================================
// GitHub Types Tests
// ============================================================================

func TestGitHubRelease_Serialization(t *testing.T) {
	release := GitHubRelease{
		TagName: "v1.0.0",
		Assets: []GitHubAsset{
			{Name: "codex-linux-x64", BrowserDownloadURL: "https://example.com/codex-linux-x64"},
			{Name: "codex-darwin-arm64", BrowserDownloadURL: "https://example.com/codex-darwin-arm64"},
		},
	}

	data, err := json.Marshal(release)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GitHubRelease
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.TagName != "v1.0.0" {
		t.Error("TagName mismatch")
	}
	if len(decoded.Assets) != 2 {
		t.Error("Assets length mismatch")
	}
}

func TestGitHubAsset_Serialization(t *testing.T) {
	asset := GitHubAsset{
		Name:               "codex-linux-x64",
		BrowserDownloadURL: "https://github.com/openai/codex/releases/download/v1.0.0/codex-linux-x64",
	}

	data, err := json.Marshal(asset)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if !strings.Contains(string(data), "codex-linux-x64") {
		t.Error("JSON should contain asset name")
	}
	if !strings.Contains(string(data), "browser_download_url") {
		t.Error("JSON should contain download URL")
	}
}

// ============================================================================
// getCacheDir Tests
// ============================================================================

func TestGetCacheDir_Packager(t *testing.T) {
	cacheDir, err := getCacheDir()
	if err != nil {
		t.Fatalf("getCacheDir failed: %v", err)
	}

	if cacheDir == "" {
		t.Error("Cache directory should not be empty")
	}

	// Should contain lurus-switch
	if !strings.Contains(cacheDir, "lurus-switch") {
		t.Error("Cache directory should contain 'lurus-switch'")
	}

	// Directory should be created
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("Cache directory should be created")
	}
}

func TestGetCacheDir_PlatformSpecific(t *testing.T) {
	cacheDir, err := getCacheDir()
	if err != nil {
		t.Fatalf("getCacheDir failed: %v", err)
	}

	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" && !strings.HasPrefix(cacheDir, localAppData) {
			t.Error("Windows cache should be under LOCALAPPDATA")
		}
	case "darwin":
		if !strings.Contains(cacheDir, "Library/Caches") {
			t.Error("macOS cache should be under ~/Library/Caches")
		}
	case "linux":
		xdgCache := os.Getenv("XDG_CACHE_HOME")
		if xdgCache != "" {
			if !strings.HasPrefix(cacheDir, xdgCache) {
				t.Error("Linux cache should respect XDG_CACHE_HOME")
			}
		} else {
			if !strings.Contains(cacheDir, ".cache") {
				t.Error("Linux cache should be under ~/.cache")
			}
		}
	}
}

// ============================================================================
// DownloadCodex Tests (with mock server)
// ============================================================================

func TestRustPackager_DownloadCodex_CacheHit(t *testing.T) {
	tmpDir := t.TempDir()
	p := &RustPackager{cacheDir: tmpDir}

	// Create a cached binary
	version := "v1.0.0"
	assetName := p.getAssetName()
	if assetName == "" {
		t.Skip("Unsupported platform")
	}

	cachedPath := filepath.Join(tmpDir, "codex", version, assetName)
	if err := os.MkdirAll(filepath.Dir(cachedPath), 0755); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}
	if err := os.WriteFile(cachedPath, []byte("cached binary"), 0755); err != nil {
		t.Fatalf("Failed to create cached file: %v", err)
	}

	// Should return cached path without making HTTP request
	result, err := p.DownloadCodex(version)
	if err != nil {
		t.Fatalf("DownloadCodex failed: %v", err)
	}

	if result != cachedPath {
		t.Errorf("Expected cached path '%s', got '%s'", cachedPath, result)
	}
}

func TestRustPackager_DownloadCodex_WithMockServer(t *testing.T) {
	// Create a mock GitHub API server
	assetName := "codex-test-binary"
	binaryContent := "mock binary content"

	mux := http.NewServeMux()

	// Mock the release API
	mux.HandleFunc("/repos/openai/codex/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		release := GitHubRelease{
			TagName: "v1.0.0",
			Assets: []GitHubAsset{
				{
					Name:               assetName,
					BrowserDownloadURL: "mock://download/" + assetName,
				},
			},
		}
		json.NewEncoder(w).Encode(release)
	})

	// Mock the binary download
	mux.HandleFunc("/download/"+assetName, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(binaryContent))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Note: This test is limited because we can't inject the API URL into DownloadCodex
	// The actual DownloadCodex uses hardcoded GitHub URLs
	// This test verifies the mock server works, but can't be used for actual testing
	t.Skip("Cannot inject mock server URL into DownloadCodex")
}

func TestRustPackager_DownloadCodex_UnsupportedPlatform(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a packager that will return empty asset name
	p := &RustPackager{cacheDir: tmpDir}

	assetName := p.getAssetName()
	if assetName == "" {
		// This platform is unsupported, test the error path
		_, err := p.DownloadCodex("v1.0.0")
		if err == nil {
			t.Error("Expected error for unsupported platform")
		}
		if !strings.Contains(err.Error(), "unsupported platform") {
			t.Error("Error should mention unsupported platform")
		}
	} else {
		t.Skip("Platform is supported, cannot test unsupported path")
	}
}

// ============================================================================
// Additional matchesAsset Tests
// ============================================================================

func TestRustPackager_MatchesAsset_CaseInsensitive(t *testing.T) {
	p := &RustPackager{}

	// Test case insensitivity
	testCases := []struct {
		name    string
		goos    string
		goarch  string
		matches bool
	}{
		{"CODEX-WINDOWS-X64.EXE", "windows", "amd64", true},
		{"Codex-Windows-X64.exe", "windows", "amd64", true},
		{"CODEX-LINUX-X64", "linux", "amd64", true},
		{"CODEX-DARWIN-ARM64", "darwin", "arm64", true},
	}

	for _, tc := range testCases {
		if runtime.GOOS != tc.goos || runtime.GOARCH != tc.goarch {
			continue
		}
		result := p.matchesAsset(tc.name)
		if result != tc.matches {
			t.Errorf("matchesAsset(%s) = %v, expected %v", tc.name, result, tc.matches)
		}
	}
}

func TestRustPackager_MatchesAsset_AMD64Variants(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("Skipping AMD64-specific test")
	}

	p := &RustPackager{}

	// All these should match AMD64 on current OS
	variants := []string{"x64", "amd64", "x86_64"}
	for _, variant := range variants {
		name := fmt.Sprintf("codex-%s-%s", strings.ToLower(runtime.GOOS), variant)
		if runtime.GOOS == "darwin" {
			// Darwin needs special handling
			for _, osName := range []string{"darwin", "macos", "apple"} {
				testName := fmt.Sprintf("codex-%s-%s", osName, variant)
				if !p.matchesAsset(testName) {
					t.Errorf("Expected %s to match AMD64 on darwin", testName)
				}
			}
		} else if p.matchesAsset(name) == false && runtime.GOOS != "windows" {
			// Windows needs .exe suffix
			t.Errorf("Expected %s to match AMD64", name)
		}
	}
}

func TestRustPackager_MatchesAsset_ARM64Variants(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("Skipping ARM64-specific test")
	}

	p := &RustPackager{}

	// All these should match ARM64
	variants := []string{"arm64", "aarch64"}
	for _, variant := range variants {
		name := fmt.Sprintf("codex-%s-%s", strings.ToLower(runtime.GOOS), variant)
		if runtime.GOOS == "darwin" {
			for _, osName := range []string{"darwin", "macos", "apple"} {
				testName := fmt.Sprintf("codex-%s-%s", osName, variant)
				if !p.matchesAsset(testName) {
					t.Errorf("Expected %s to match ARM64 on darwin", testName)
				}
			}
		} else if !p.matchesAsset(name) {
			t.Errorf("Expected %s to match ARM64", name)
		}
	}
}

// ============================================================================
// Bun/Node Packager Constructor Tests
// ============================================================================

func TestNewBunPackager(t *testing.T) {
	p, err := NewBunPackager()
	if err != nil {
		// Bun not installed, which is fine
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Unexpected error: %v", err)
		}
		return
	}

	if p == nil {
		t.Fatal("NewBunPackager should return non-nil packager when Bun is installed")
	}

	if p.bunPath == "" {
		t.Error("bunPath should not be empty")
	}
}

func TestNewNodePackager(t *testing.T) {
	p, err := NewNodePackager()
	if err != nil {
		// Node not installed, which is fine
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Unexpected error: %v", err)
		}
		return
	}

	if p == nil {
		t.Fatal("NewNodePackager should return non-nil packager when Node is installed")
	}

	if p.bunPath == "" && p.npxPath == "" {
		t.Error("bunPath or npxPath should not be empty")
	}
}

// ============================================================================
// Wrapper Content Validation Tests
// ============================================================================

func TestBunPackager_GenerateWrapper_Content(t *testing.T) {
	p := &BunPackager{bunPath: "/usr/local/bin/bun"}

	wrapper := p.generateWrapper("/test/config")

	// Verify essential parts
	requiredParts := []string{
		"import { spawn }",
		"import { existsSync, readFileSync }",
		"import { join }",
		"const CONFIG_DIR",
		"settings.json",
		"CLAUDE_CONFIG_DIR",
		"spawn(\"claude\"",
		"process.exit",
	}

	for _, part := range requiredParts {
		if !strings.Contains(wrapper, part) {
			t.Errorf("Wrapper should contain '%s'", part)
		}
	}
}

func TestNodePackager_GenerateWrapper_Content(t *testing.T) {
	p := &NodePackager{bunPath: "/usr/local/bin/bun", npxPath: "/usr/local/bin/npx"}

	wrapper := p.generateWrapper("/test/config")

	// Verify essential parts
	requiredParts := []string{
		"require(\"child_process\")",
		"require(\"fs\")",
		"require(\"path\")",
		"const CONFIG_DIR",
		"GEMINI.md",
		"gemini-settings.json",
		"GEMINI_CONFIG_DIR",
		"spawn(\"gemini\"",
		"process.exit",
	}

	for _, part := range requiredParts {
		if !strings.Contains(wrapper, part) {
			t.Errorf("Wrapper should contain '%s'", part)
		}
	}
}

// ============================================================================
// Edge Case Tests
// ============================================================================

func TestBunPackager_GenerateWrapper_SpecialChars(t *testing.T) {
	p := &BunPackager{bunPath: "/usr/local/bin/bun"}

	// Test path with spaces
	wrapper := p.generateWrapper("/path with spaces/config")
	if !strings.Contains(wrapper, "/path with spaces/config") {
		t.Error("Wrapper should handle paths with spaces")
	}
}

func TestNodePackager_GenerateWrapper_SpecialChars(t *testing.T) {
	p := &NodePackager{bunPath: "bun", npxPath: "npx"}

	// Test path with spaces (non-Windows)
	if runtime.GOOS != "windows" {
		wrapper := p.generateWrapper("/path with spaces/config")
		if !strings.Contains(wrapper, "/path with spaces/config") {
			t.Error("Wrapper should handle paths with spaces")
		}
	}
}

func TestRustPackager_DownloadFile_NetworkError(t *testing.T) {
	tmpDir := t.TempDir()
	p := &RustPackager{cacheDir: tmpDir}

	// Use invalid URL to trigger network error
	err := p.downloadFile("http://invalid.localhost.test:99999/file", filepath.Join(tmpDir, "file"))
	if err == nil {
		t.Error("Expected network error")
	}
}

func TestRustPackager_DownloadFile_CreateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	p := &RustPackager{cacheDir: tmpDir}

	// Try to write to a directory (should fail)
	dirPath := filepath.Join(tmpDir, "isdir")
	os.MkdirAll(dirPath, 0755)

	err := p.downloadFile(server.URL, dirPath)
	if err == nil {
		t.Error("Expected error when destination is a directory")
	}
}

// ============================================================================
// Package Method Tests (with mock dependencies)
// ============================================================================

func TestRustPackager_Package_OutputDirCreation(t *testing.T) {
	tmpDir := t.TempDir()
	p := &RustPackager{cacheDir: tmpDir}

	// Create a cached binary
	version := "v1.0.0"
	assetName := p.getAssetName()
	if assetName == "" {
		t.Skip("Unsupported platform")
	}

	cachedPath := filepath.Join(tmpDir, "codex", version, assetName)
	if err := os.MkdirAll(filepath.Dir(cachedPath), 0755); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}
	if err := os.WriteFile(cachedPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("Failed to create cached file: %v", err)
	}

	// Package to nested output directory
	outputDir := filepath.Join(tmpDir, "output", "nested", "dir")
	configDir := filepath.Join(tmpDir, "config")
	os.MkdirAll(configDir, 0755)

	result, err := p.Package(configDir, outputDir, version)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}

	// Verify output directory was created
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("Output directory should be created")
	}

	// Verify binary was copied
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Error("Binary should be copied to output")
	}
}

func TestRustPackager_CopyFile_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	p := &RustPackager{cacheDir: tmpDir}

	// Create a larger file (1MB)
	srcPath := filepath.Join(tmpDir, "large.bin")
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	if err := os.WriteFile(srcPath, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Copy file
	dstPath := filepath.Join(tmpDir, "large_copy.bin")
	err := p.copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify size
	srcInfo, _ := os.Stat(srcPath)
	dstInfo, _ := os.Stat(dstPath)
	if srcInfo.Size() != dstInfo.Size() {
		t.Errorf("File size mismatch: src=%d, dst=%d", srcInfo.Size(), dstInfo.Size())
	}
}

