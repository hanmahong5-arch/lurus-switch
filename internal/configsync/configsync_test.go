package configsync

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readAllEntries decompresses a bundle and returns a name->content map, so
// content assertions inspect the actual file bytes rather than the
// compressed zip stream (where no literal substring would ever match).
func readAllEntries(t *testing.T, data []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]string)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b := new(bytes.Buffer)
		if _, err := b.ReadFrom(rc); err != nil {
			t.Fatal(err)
		}
		rc.Close()
		out[f.Name] = b.String()
	}
	return out
}

// allEntriesJoined concatenates every decompressed entry so a single
// "no secret anywhere" assertion can scan the whole bundle.
func allEntriesJoined(t *testing.T, data []byte) string {
	t.Helper()
	var sb strings.Builder
	for _, v := range readAllEntries(t, data) {
		sb.WriteString(v)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// buildBundleWithSchema produces a minimal zip whose manifest declares the
// given schema version — used to exercise the importer's version guard.
func buildBundleWithSchema(t *testing.T, version int) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	mf := Manifest{SchemaVersion: version, Components: []string{}}
	data, _ := json.Marshal(mf)
	fw, err := zw.Create(manifestEntry)
	if err != nil {
		t.Fatal(err)
	}
	fw.Write(data)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// fixture builds a Dirs pair with a populated AppData + Home so Export has
// something to capture.
func fixture(t *testing.T) Dirs {
	t.Helper()
	root := t.TempDir()
	appData := filepath.Join(root, "appdata")
	home := filepath.Join(root, "home")
	mustWrite(t, filepath.Join(appData, "app-settings.json"), `{"theme":"dark","reseller":{"adminToken":"secret-admin"}}`)
	mustWrite(t, filepath.Join(appData, "custom-providers.json"), `[{"id":"c1","name":"X","baseUrl":"https://x.test","apiKeyB64":"c2stc2VjcmV0"}]`)
	mustWrite(t, filepath.Join(appData, "snapshots", "snap1.json"), `{"id":"snap1"}`)
	mustWrite(t, filepath.Join(appData, "prompts", "p1.json"), `{"id":"p1"}`)
	mustWrite(t, filepath.Join(appData, "mcp-presets", "m1.json"), `{"id":"m1"}`)
	mustWrite(t, filepath.Join(home, ".claude", "settings.json"), `{"env":{"ANTHROPIC_API_KEY":"sk-ant-leak"},"model":"opus"}`)
	mustWrite(t, filepath.Join(home, ".codex", "config.toml"), "model = \"o3\"\napi_key = \"sk-codex-leak\"\n")
	return Dirs{AppData: appData, Home: home}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestExport_ManifestListsPresentComponents(t *testing.T) {
	d := fixture(t)
	var buf bytes.Buffer
	mf, err := Export(d, true, "1.2.3", &buf)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		CompAppSettings: true, CompCustomProviders: true, CompToolConfigs: true,
		CompMCPPresets: true, CompPrompts: true, CompSnapshots: true,
	}
	for _, c := range mf.Components {
		delete(want, c)
	}
	if len(want) != 0 {
		t.Errorf("manifest missing components: %v", want)
	}
	if mf.AppVersion != "1.2.3" || mf.SchemaVersion != SchemaVersion {
		t.Errorf("manifest meta wrong: %+v", mf)
	}
}

func TestExport_RedactsSecretsWhenKeysExcluded(t *testing.T) {
	d := fixture(t)
	var buf bytes.Buffer
	if _, err := Export(d, false, "1", &buf); err != nil {
		t.Fatal(err)
	}
	joined := allEntriesJoined(t, buf.Bytes())
	for _, leak := range []string{"secret-admin", "sk-ant-leak", "sk-codex-leak", "c2stc2VjcmV0"} {
		if strings.Contains(joined, leak) {
			t.Errorf("redacted export still contains secret %q", leak)
		}
	}
}

func TestExport_IncludesSecretsWhenRequested(t *testing.T) {
	d := fixture(t)
	var buf bytes.Buffer
	if _, err := Export(d, true, "1", &buf); err != nil {
		t.Fatal(err)
	}
	joined := allEntriesJoined(t, buf.Bytes())
	if !strings.Contains(joined, "sk-ant-leak") {
		t.Error("includeKeys=true should preserve the API key")
	}
}

func TestRoundTrip_ExportThenImport(t *testing.T) {
	src := fixture(t)
	var buf bytes.Buffer
	if _, err := Export(src, true, "1", &buf); err != nil {
		t.Fatal(err)
	}

	// Fresh, empty destination.
	root := t.TempDir()
	dst := Dirs{AppData: filepath.Join(root, "appdata"), Home: filepath.Join(root, "home")}

	written, err := Apply(bytes.NewReader(buf.Bytes()), dst, map[string]bool{
		CompAppSettings: true, CompCustomProviders: true, CompToolConfigs: true,
		CompMCPPresets: true, CompPrompts: true, CompSnapshots: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 6 {
		t.Errorf("expected 6 components written, got %d (%v)", len(written), written)
	}

	// Spot-check a couple files round-tripped byte-identically.
	got, _ := os.ReadFile(filepath.Join(dst.AppData, "app-settings.json"))
	if !strings.Contains(string(got), "secret-admin") {
		t.Error("app-settings not restored")
	}
	if _, err := os.Stat(filepath.Join(dst.Home, ".codex", "config.toml")); err != nil {
		t.Error("tool config not restored")
	}
}

func TestApply_RespectsAcceptedSubset(t *testing.T) {
	src := fixture(t)
	var buf bytes.Buffer
	Export(src, true, "1", &buf)

	root := t.TempDir()
	dst := Dirs{AppData: filepath.Join(root, "appdata"), Home: filepath.Join(root, "home")}
	written, err := Apply(bytes.NewReader(buf.Bytes()), dst, map[string]bool{CompPrompts: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 1 || written[0] != CompPrompts {
		t.Fatalf("expected only prompts written, got %v", written)
	}
	if _, err := os.Stat(filepath.Join(dst.AppData, "app-settings.json")); !os.IsNotExist(err) {
		t.Error("app-settings should not have been written")
	}
}

func TestApply_BacksUpExistingFile(t *testing.T) {
	src := fixture(t)
	var buf bytes.Buffer
	Export(src, true, "1", &buf)

	// Destination already has an app-settings with different content.
	root := t.TempDir()
	dst := Dirs{AppData: filepath.Join(root, "appdata"), Home: filepath.Join(root, "home")}
	mustWrite(t, filepath.Join(dst.AppData, "app-settings.json"), `{"theme":"light"}`)

	if _, err := Apply(bytes.NewReader(buf.Bytes()), dst, map[string]bool{CompAppSettings: true}); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(dst.AppData, "app-settings.json.before-import-*"))
	if len(matches) == 0 {
		t.Error("expected a .before-import backup of the overwritten file")
	}
	backup, _ := os.ReadFile(matches[0])
	if !strings.Contains(string(backup), "light") {
		t.Errorf("backup should hold the prior content, got %s", backup)
	}
}

func TestPreview_ReportsActions(t *testing.T) {
	src := fixture(t)
	var buf bytes.Buffer
	Export(src, true, "1", &buf)

	root := t.TempDir()
	dst := Dirs{AppData: filepath.Join(root, "appdata"), Home: filepath.Join(root, "home")}
	mustWrite(t, filepath.Join(dst.AppData, "app-settings.json"), `{"theme":"light"}`)

	pv, err := Preview(bytes.NewReader(buf.Bytes()), dst)
	if err != nil {
		t.Fatal(err)
	}
	actions := map[string]string{}
	for _, c := range pv.Components {
		actions[c.Key] = c.Action
	}
	if actions[CompAppSettings] != "overwrite" {
		t.Errorf("app-settings action = %q, want overwrite", actions[CompAppSettings])
	}
	if actions[CompPrompts] != "create" {
		t.Errorf("prompts action = %q, want create", actions[CompPrompts])
	}
}

func TestImport_RejectsSchemaMismatch(t *testing.T) {
	// Hand-craft a bundle whose manifest claims a future schema version.
	var buf bytes.Buffer
	d := fixture(t)
	Export(d, true, "1", &buf)

	// Rewrite the manifest with a bumped schema by re-zipping is overkill;
	// instead assert the importer rejects a v2 manifest constructed inline.
	bad := buildBundleWithSchema(t, 999)
	if _, err := Preview(bytes.NewReader(bad), d); err == nil {
		t.Error("expected schema-mismatch rejection")
	}
}

func TestImport_RejectsNonZip(t *testing.T) {
	if _, err := Preview(strings.NewReader("not a zip"), Dirs{}); err == nil {
		t.Error("expected error for non-zip input")
	}
}

func TestRedactJSON_NestedAndArrays(t *testing.T) {
	in := []byte(`{"a":{"apiKey":"x"},"list":[{"openaiKey":"y"}],"keep":"ok"}`)
	out := redactJSON(in)
	var v map[string]any
	json.Unmarshal(out, &v)
	if v["a"].(map[string]any)["apiKey"] != redactedPlaceholder {
		t.Error("nested apiKey not redacted")
	}
	if v["list"].([]any)[0].(map[string]any)["openaiKey"] != redactedPlaceholder {
		t.Error("array openaiKey not redacted")
	}
	if v["keep"] != "ok" {
		t.Error("non-secret field was altered")
	}
}

func TestZipSlipGuard(t *testing.T) {
	if _, _, ok := zipPathTarget("../../etc/passwd", Dirs{AppData: "/a", Home: "/h"}); ok {
		t.Error("zip-slip path should be rejected")
	}
}
