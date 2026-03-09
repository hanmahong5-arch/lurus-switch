package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ===========================
// isNewer tests
// ===========================

func TestIsNewer_PatchVersion(t *testing.T) {
	tests := []struct {
		latest  string
		current string
		want    bool
	}{
		{"1.0.1", "1.0.0", true},
		{"1.0.0", "1.0.1", false},
		{"1.0.0", "1.0.0", false},
		{"2.0.0", "1.9.9", true},
		{"1.10.0", "1.9.0", true},
		{"1.0.10", "1.0.9", true},
	}
	for _, tc := range tests {
		got := isNewer(tc.latest, tc.current)
		if got != tc.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}

func TestIsNewer_MajorVersion(t *testing.T) {
	if !isNewer("2.0.0", "1.0.0") {
		t.Error("2.0.0 should be newer than 1.0.0")
	}
	if isNewer("1.0.0", "2.0.0") {
		t.Error("1.0.0 should not be newer than 2.0.0")
	}
}

func TestIsNewer_MinorVersion(t *testing.T) {
	if !isNewer("1.2.0", "1.1.0") {
		t.Error("1.2.0 should be newer than 1.1.0")
	}
	if isNewer("1.1.0", "1.2.0") {
		t.Error("1.1.0 should not be newer than 1.2.0")
	}
}

func TestIsNewer_SameVersion(t *testing.T) {
	if isNewer("1.2.3", "1.2.3") {
		t.Error("same version should not be newer")
	}
}

func TestIsNewer_LongerLatest(t *testing.T) {
	// more parts in latest means potentially newer when equal at common parts
	if !isNewer("1.0.0.1", "1.0.0") {
		t.Error("1.0.0.1 should be newer than 1.0.0 (more parts)")
	}
}

func TestIsNewer_ShorterLatest(t *testing.T) {
	// fewer parts: 1.0 vs 1.0.0 — equal at common parts, shorter has fewer
	if isNewer("1.0", "1.0.0") {
		t.Error("1.0 should not be newer than 1.0.0")
	}
}

func TestIsNewer_WithAlphaChars(t *testing.T) {
	// parseIntSafe extracts only digits, so "1-beta" → 1, "1.0" part → 10
	// "1.0.0-beta" splits on "." to ["1","0","0-beta"]; "0-beta" → 0
	if isNewer("1.0.0-beta", "1.0.0") {
		t.Error("1.0.0-beta should not be newer than 1.0.0")
	}
}

func TestIsNewer_LargeNumbers(t *testing.T) {
	if !isNewer("10.0.0", "9.0.0") {
		t.Error("10.0.0 should be newer than 9.0.0")
	}
	if !isNewer("1.100.0", "1.99.0") {
		t.Error("1.100.0 should be newer than 1.99.0")
	}
}

func TestIsNewer_EmptyStrings(t *testing.T) {
	// Both empty parts → 0 vs 0, not newer
	if isNewer("", "") {
		t.Error("empty vs empty should not be newer")
	}
}

// ===========================
// parseIntSafe tests
// ===========================

func TestParseIntSafe_Normal(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"0", 0},
		{"1", 1},
		{"42", 42},
		{"100", 100},
		{"1234567", 1234567},
	}
	for _, tc := range tests {
		got := parseIntSafe(tc.input)
		if got != tc.want {
			t.Errorf("parseIntSafe(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestParseIntSafe_NonNumeric(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"abc", 0},
		{"v2", 2},
		{"rc1", 1},
		{"1-beta", 1},
	}
	for _, tc := range tests {
		got := parseIntSafe(tc.input)
		if got != tc.want {
			t.Errorf("parseIntSafe(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestParseIntSafe_Zero(t *testing.T) {
	if got := parseIntSafe("0"); got != 0 {
		t.Errorf("parseIntSafe(0) = %d, want 0", got)
	}
}

func TestParseIntSafe_AllDigits(t *testing.T) {
	if got := parseIntSafe("9"); got != 9 {
		t.Errorf("parseIntSafe(9) = %d, want 9", got)
	}
}

// ===========================
// CheckAllTools offline test
// ===========================

func TestCheckAllTools_EmptyVersions(t *testing.T) {
	// With no versions provided, CheckAllTools should still return results for the 3 npm tools
	checker := NewNpmChecker()
	// We don't actually hit the network here because the empty map means
	// no versions are looked up — but internally it still calls CheckUpdate.
	// Since we can't guarantee network access in tests, just verify it doesn't panic.
	results := checker.CheckAllTools(map[string]string{})
	if results == nil {
		t.Error("CheckAllTools should return non-nil map")
	}
}

// ===========================
// CheckUpdate with mock HTTP server
// ===========================

func TestCheckUpdate_UpdateAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(npmPackageInfo{Version: "2.5.0"})
	}))
	defer srv.Close()

	checker := &NpmChecker{client: srv.Client()}
	info, err := checkUpdateWithBaseURL(checker, srv.URL, "test-pkg", "1.0.0")
	if err != nil {
		t.Fatalf("checkUpdate error: %v", err)
	}
	if info.LatestVersion != "2.5.0" {
		t.Errorf("LatestVersion = %q, want 2.5.0", info.LatestVersion)
	}
	if !info.UpdateAvailable {
		t.Error("UpdateAvailable should be true since 2.5.0 > 1.0.0")
	}
}

func TestCheckUpdate_NoUpdate_SameVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(npmPackageInfo{Version: "1.0.0"})
	}))
	defer srv.Close()

	checker := &NpmChecker{client: srv.Client()}
	info, err := checkUpdateWithBaseURL(checker, srv.URL, "test-pkg", "1.0.0")
	if err != nil {
		t.Fatalf("checkUpdate error: %v", err)
	}
	if info.UpdateAvailable {
		t.Error("UpdateAvailable should be false when versions are equal")
	}
}

func TestCheckUpdate_NoUpdate_Downgrade(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(npmPackageInfo{Version: "0.9.0"})
	}))
	defer srv.Close()

	checker := &NpmChecker{client: srv.Client()}
	info, err := checkUpdateWithBaseURL(checker, srv.URL, "test-pkg", "1.0.0")
	if err != nil {
		t.Fatalf("checkUpdate error: %v", err)
	}
	if info.UpdateAvailable {
		t.Error("UpdateAvailable should be false when current is newer")
	}
}

func TestCheckUpdate_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	checker := &NpmChecker{client: srv.Client()}
	_, err := checkUpdateWithBaseURL(checker, srv.URL, "missing-pkg", "1.0.0")
	if err == nil {
		t.Error("expected error for HTTP 404")
	}
}

func TestCheckUpdate_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{not valid json"))
	}))
	defer srv.Close()

	checker := &NpmChecker{client: srv.Client()}
	_, err := checkUpdateWithBaseURL(checker, srv.URL, "test-pkg", "1.0.0")
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestCheckUpdate_EmptyCurrentVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(npmPackageInfo{Version: "2.0.0"})
	}))
	defer srv.Close()

	checker := &NpmChecker{client: srv.Client()}
	// Empty current version → UpdateAvailable should be false (can't compare)
	info, err := checkUpdateWithBaseURL(checker, srv.URL, "test-pkg", "")
	if err != nil {
		t.Fatalf("checkUpdate error: %v", err)
	}
	if info.UpdateAvailable {
		t.Error("UpdateAvailable should be false when currentVersion is empty")
	}
}

// checkUpdateWithBaseURL duplicates CheckUpdate logic using a custom base URL for testing
func checkUpdateWithBaseURL(n *NpmChecker, baseURL, packageName, currentVersion string) (*UpdateInfo, error) {
	reqURL := fmt.Sprintf("%s/%s/latest", baseURL, packageName)
	resp, err := n.client.Get(reqURL)
	if err != nil {
		return &UpdateInfo{
			Name:            packageName,
			CurrentVersion:  currentVersion,
			LatestVersion:   "unknown (offline)",
			UpdateAvailable: false,
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry returned HTTP %d for package %s", resp.StatusCode, packageName)
	}

	var info npmPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse npm registry response: %w", err)
	}

	return &UpdateInfo{
		Name:            packageName,
		CurrentVersion:  currentVersion,
		LatestVersion:   info.Version,
		UpdateAvailable: info.Version != "" && currentVersion != "" && info.Version != currentVersion && isNewer(info.Version, currentVersion),
	}, nil
}
