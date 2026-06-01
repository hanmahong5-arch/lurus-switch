package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"lurus-switch/internal/appconfig"
	"lurus-switch/internal/whitelabel"
)

// ============================
// White-label packager (S-Xc.1 + S-Xc.2) — Reseller-facing build pipeline.
// ============================
//
// The Reseller workflow:
//
//   1. PackagerPage collects brand inputs (name, hub URL, color, logo).
//   2. ResolveBaseBinaryPath() points at the running Switch.exe; the
//      Reseller can also override to a fresher build.
//   3. BuildWhiteLabelPackage() runs the packer, returning paths +
//      hashes. UI surfaces the output dir and a "open in explorer" link.
//
// The HMAC key would normally come from the Hub admin endpoint
// (/api/v2/admin/whitelabel/hmac-key — TBD). Until that ships, we derive
// a stable per-Hub key from the saved AdminToken so each Reseller's
// builds are still cross-verifiable on the EndUser side. The deferred
// migration is straightforward: swap the key source, no API change.

// WhiteLabelInputs is the JSON payload accepted from the frontend. Mirrors
// whitelabel.Profile but uses the fields the UI actually surfaces — keep
// the binding boundary stable even when the underlying Profile grows.
type WhiteLabelInputs struct {
	BrandName      string `json:"brandName"`
	HubURL         string `json:"hubUrl"`
	TenantSlug     string `json:"tenantSlug,omitempty"`
	PrimaryColor   string `json:"primaryColor,omitempty"`
	LogoBase64     string `json:"logoBase64,omitempty"`
	SupportContact string `json:"supportContact,omitempty"`
	OutputDir      string `json:"outputDir,omitempty"`
	IconPath       string `json:"iconPath,omitempty"`
}

// WhiteLabelOutput is the BuildResult re-spelled in camelCase so the
// frontend doesn't have to rename keys.
type WhiteLabelOutput struct {
	OutputDir     string   `json:"outputDir"`
	BinaryPath    string   `json:"binaryPath"`
	SidecarPath   string   `json:"sidecarPath"`
	BinarySHA256  string   `json:"binarySha256"`
	SidecarSHA256 string   `json:"sidecarSha256"`
	Notes         []string `json:"notes,omitempty"`
}

// BuildWhiteLabelPackage produces a branded EndUser distribution from
// the running Switch binary plus the Reseller's brand inputs.
//
// The built artifacts land in OutputDir (or `<appdata>/lurus-switch/
// whitelabel-builds/<brand-slug>` when empty). On success, the UI gets
// paths + hashes back; the Reseller can ship the directory contents as
// a ZIP or installer payload.
func (a *App) BuildWhiteLabelPackage(in WhiteLabelInputs) (*WhiteLabelOutput, error) {
	// Empty id → bus auto-generates a fresh per-run id so the activity
	// pane shows distinct entries for repeated builds.
	op := a.activityBus.Op("", "构建白标安装包", "Building white-label package")

	op.Progress("解析底包路径", "Resolving base binary", 10, 4, 1)
	base, err := resolveBaseBinaryPath()
	if err != nil {
		op.Error(err.Error())
		return nil, err
	}

	outDir := strings.TrimSpace(in.OutputDir)
	if outDir == "" {
		outDir = defaultBuildOutputDir(in.BrandName)
	}

	op.Progress("准备签名密钥", "Preparing HMAC key", 25, 4, 2)
	key, err := whitelabelHMACKey()
	if err != nil {
		op.Error(err.Error())
		return nil, err
	}

	op.Progress("打包二进制 + 资源", "Packing binary + assets", 50, 4, 3)
	res, err := whitelabel.Build(whitelabel.BuildOpts{
		Profile: whitelabel.Profile{
			BrandName:      strings.TrimSpace(in.BrandName),
			HubURL:         strings.TrimSpace(in.HubURL),
			TenantSlug:     strings.TrimSpace(in.TenantSlug),
			PrimaryColor:   strings.TrimSpace(in.PrimaryColor),
			LogoBase64:     in.LogoBase64,
			SupportContact: strings.TrimSpace(in.SupportContact),
		},
		HMACKey:        key,
		BaseBinaryPath: base,
		OutputDir:      outDir,
		IconPath:       in.IconPath,
	})
	if err != nil {
		op.Error(err.Error())
		return nil, fmt.Errorf("build white-label package: %w", err)
	}
	op.Done("打包完成 — "+res.BinaryPath, "Package built — "+res.BinaryPath)
	appendBuildHistory(BuildHistoryEntry{
		BuiltAt:    time.Now(),
		BrandName:  strings.TrimSpace(in.BrandName),
		HubURL:     strings.TrimSpace(in.HubURL),
		BinaryPath: res.BinaryPath,
		SHA256:     res.SHA256,
	})
	return &WhiteLabelOutput{
		OutputDir:     res.OutputDir,
		BinaryPath:    res.BinaryPath,
		SidecarPath:   res.SidecarPath,
		BinarySHA256:  res.SHA256,
		SidecarSHA256: res.SidecarSHA256,
		Notes:         res.Notes,
	}, nil
}

// PreviewWhiteLabelLogo decodes the operator-supplied logo to surface
// useful metadata (size + content-type detection) so the UI can warn
// before running a full Build. Pure validation — no FS writes.
func (a *App) PreviewWhiteLabelLogo(logoBase64 string) (map[string]any, error) {
	if logoBase64 == "" {
		return nil, errors.New("logo is empty")
	}
	raw, err := base64.StdEncoding.DecodeString(logoBase64)
	if err != nil {
		return nil, fmt.Errorf("not valid base64: %w", err)
	}
	mime := "application/octet-stream"
	switch {
	case len(raw) >= 8 && string(raw[:8]) == "\x89PNG\r\n\x1a\n":
		mime = "image/png"
	case len(raw) >= 4 && (string(raw[:4]) == "GIF8"):
		mime = "image/gif"
	case len(raw) >= 3 && raw[0] == 0xFF && raw[1] == 0xD8 && raw[2] == 0xFF:
		mime = "image/jpeg"
	case len(raw) >= 5 && string(raw[:5]) == "<?xml":
		mime = "image/svg+xml"
	}
	return map[string]any{
		"size":      len(raw),
		"mime":      mime,
		"limit":     whitelabel.MaxLogoBytes,
		"oversized": len(raw) > whitelabel.MaxLogoBytes,
	}, nil
}

// resolveBaseBinaryPath finds the source Switch exe to clone. Currently
// uses os.Executable() — the running binary. Future iteration may add
// "download from GitHub releases" once the auto-update channel is wired.
func resolveBaseBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate running binary: %w", err)
	}
	return exe, nil
}

// defaultBuildOutputDir produces a per-brand directory under appdata.
// Keeps successive builds tidy and discoverable from the file manager.
func defaultBuildOutputDir(brand string) string {
	base := appDataBaseDir()
	slug := strings.ToLower(strings.TrimSpace(brand))
	if slug == "" {
		slug = "untitled"
	}
	// Reuse the packer's slug rules indirectly via filename safety.
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, slug)
	return filepath.Join(base, "whitelabel-builds", slug)
}

// whitelabelHMACKey resolves the per-Hub HMAC key used to sign + verify
// the white-label sidecar. The resolver tries two sources in order:
//
//  1. Hub endpoint `/api/v2/admin/whitelabel/hmac-key?tenant_slug=<slug>`
//     (Track 2.3 / Phase D). When the Reseller has configured HubURL +
//     AdminToken + TenantSlug, we fetch the per-tenant key from there.
//     The Hub-issued key rotates server-side and lets the platform
//     revoke a Reseller's signing capability without shipping a new
//     Switch build.
//  2. Hardcoded `whitelabelBuildSecret` fallback. Used when the Hub
//     endpoint is absent (older Hub deployments), the Reseller hasn't
//     finished setup, or any transport error trips the remote call.
//     This is also the only path on a freshly-installed EndUser binary
//     where there is no Reseller config at all.
//
// 🔑 KEY DERIVATION HISTORY (audit blocker #1, 2026-05): the original
// implementation derived the key from `s.Reseller.AdminToken`, which is
// empty on a fresh EndUser machine. Result: every distributed white-
// label binary failed sidecar verification on first launch and silently
// fell through to the unset-mode picker — completely defeating the
// white-label lock. We then switched to a baked-in build secret. As of
// Phase D (2026-05), the Hub also publishes a per-tenant key so older
// builds stay compatible while new builds can be invalidated by a Hub-
// side rotation. The HMAC's purpose is tamper-detection during
// distribution (CDN, archive manipulation), NOT defense against
// malicious Resellers (who hold the secret too — the source is open).
// For that, ship signed installers (Authenticode on Windows, codesign
// on macOS) — see the manual distribution checklist in PackagerPage.
const whitelabelBuildSecret = "lurus-switch-whitelabel-v1-46de2f01-tamper-detection"

// errHubHMACKeyEndpointAbsent signals that the Hub does not expose the
// per-tenant HMAC-key endpoint (404, 405) or the call failed in a way
// that should cause the caller to silently fall back to the hardcoded
// build secret. The error is a sentinel — callers compare with
// errors.Is for the fallback decision.
var errHubHMACKeyEndpointAbsent = errors.New("hub: whitelabel hmac-key endpoint absent")

// hubHMACKeyHTTPClient is the seam tests use to swap in an httptest
// server's client. Production callers leave this nil and get a sane
// default with a short timeout (the Hub call is on the critical path
// of `BuildWhiteLabelPackage`, so the budget must be small).
var hubHMACKeyHTTPClient *http.Client

func defaultHubHMACKeyClient() *http.Client {
	if hubHMACKeyHTTPClient != nil {
		return hubHMACKeyHTTPClient
	}
	return &http.Client{Timeout: 5 * time.Second}
}

// fetchHubHMACKey calls the Hub admin endpoint and returns the 32-byte
// per-tenant HMAC key. The Hub returns `{success, data:{hmac_key:"<hex>"}}`
// and we decode the hex into raw bytes. Errors are classified into:
//
//   - errHubHMACKeyEndpointAbsent — 404 / 405 / 501 / connection refused.
//     Caller should silently fall back to the hardcoded secret so older
//     Hub deployments keep working.
//   - generic error — anything else (200 success=false, non-JSON body,
//     wrong-length hex, invalid hex). Caller still falls back, but the
//     message is preserved for stderr logging because these signal a
//     misconfigured Hub rather than an absent feature.
//
// `ctx` carries the per-call deadline. The Authorization header uses the
// same convention as internal/hub/admin.Client: raw token, no `Bearer`
// prefix — Hub middleware accepts that form for native admin tokens.
func fetchHubHMACKey(ctx context.Context, hubBaseURL, tenantSlug, adminToken string) ([]byte, error) {
	hubBaseURL = strings.TrimRight(strings.TrimSpace(hubBaseURL), "/")
	tenantSlug = strings.TrimSpace(tenantSlug)
	adminToken = strings.TrimSpace(adminToken)
	if hubBaseURL == "" {
		return nil, fmt.Errorf("%w: empty hub URL", errHubHMACKeyEndpointAbsent)
	}
	if tenantSlug == "" {
		return nil, fmt.Errorf("%w: empty tenant slug", errHubHMACKeyEndpointAbsent)
	}
	if adminToken == "" {
		return nil, fmt.Errorf("%w: empty admin token", errHubHMACKeyEndpointAbsent)
	}

	q := url.Values{}
	q.Set("tenant_slug", tenantSlug)
	endpoint := hubBaseURL + "/api/v2/admin/whitelabel/hmac-key?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build hmac-key request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", adminToken)

	resp, err := defaultHubHMACKeyClient().Do(req)
	if err != nil {
		// Connection refused / DNS failure / context deadline → treat
		// as endpoint-absent so we silently fall back. The underlying
		// transport error stays wrapped for log surfaces.
		return nil, fmt.Errorf("%w: transport: %v", errHubHMACKeyEndpointAbsent, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound ||
		resp.StatusCode == http.StatusMethodNotAllowed ||
		resp.StatusCode == http.StatusNotImplemented {
		return nil, fmt.Errorf("%w: HTTP %d", errHubHMACKeyEndpointAbsent, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read hmac-key response: %w", err)
	}

	var env struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			HMACKey string `json:"hmac_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("malformed hmac-key response (HTTP %d): %w", resp.StatusCode, err)
	}
	if !env.Success {
		msg := env.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("hub hmac-key error: %s", msg)
	}

	hexKey := strings.TrimSpace(env.Data.HMACKey)
	if len(hexKey) != 2*sha256.Size {
		// sha256.Size = 32 bytes → 64 hex chars. Anything else is a
		// Hub bug; better to scream than to silently produce a wrong-
		// shaped key that cascades into Verify failures later.
		return nil, fmt.Errorf("malformed hmac-key: expected %d hex chars, got %d", 2*sha256.Size, len(hexKey))
	}
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("malformed hmac-key: invalid hex: %w", err)
	}
	return keyBytes, nil
}

// whitelabelHMACKey returns the HMAC key for sidecar sign/verify.
//
// The key is always derived from the baked whitelabelBuildSecret via
// SHA-256. This ensures signing (BuildWhiteLabelPackage on the Reseller
// machine) and verification (applyWhiteLabelSidecar on a fresh EndUser
// machine) always use the same key regardless of which machine config
// is present.
//
// Why not the Hub per-tenant endpoint? The Hub endpoint approach
// (fetchHubHMACKey) was previously tried as a "first preference" but
// created an asymmetry: a Reseller with full Hub credentials signed with
// the Hub-issued key, while the EndUser (no Hub credentials) always fell
// back to the baked secret — so every white-label sidecar failed HMAC
// verification on first EndUser launch. The HMAC threat model is tamper-
// detection during distribution (CDN / archive manipulation), not defense
// against a determined adversary who holds the binary. For that, use
// Authenticode (Windows) or codesign (macOS) as documented in the
// PackagerPage distribution checklist.
//
// The function signature is kept as ([]byte, error) for future
// extensibility (e.g. a --strict build flag that hard-fails when the
// baked secret is missing or zero-length); callers must still check err.
func whitelabelHMACKey() ([]byte, error) {
	sum := sha256.Sum256([]byte(whitelabelBuildSecret))
	return sum[:], nil
}

// applyWhiteLabelSidecar runs at app startup. When a signed
// whitelabel.json sits next to the running exe, it locks the app to
// EndUser mode + the embedded Hub URL. Idempotent — already-locked
// installs short-circuit before doing any FS writes.
//
// Failure modes:
//
//   - No sidecar present → silent no-op (this isn't a white-label build).
//   - Sidecar present but verification fails → log + abort startup mode
//     write. EndUser code paths will see no LockedHubURL and surface a
//     clear "白标包损坏" error rather than dialing home.
//
// HMAC key resolution at startup uses the *previously-saved* AdminToken,
// since we have no Hub session yet. Distributors who rotate their token
// must re-pack — that's the documented contract.
func (a *App) applyWhiteLabelSidecar() {
	path := whitelabel.FindSidecarPath()
	if path == "" {
		return
	}
	key, err := whitelabelHMACKey()
	if err != nil {
		// No saved AdminToken yet — first launch on a fresh white-label
		// install can't verify until we know the key. Surface as a
		// startup warning; EndUser activation page will still work via
		// AppSettings.LockedHubURL once it's set elsewhere.
		fmt.Fprintf(os.Stderr, "whitelabel: skipping sidecar (key unavailable): %v\n", err)
		return
	}
	loader := &whitelabel.Loader{HMACKey: key}
	prof, err := loader.Load(path)
	if err != nil {
		if whitelabel.IsNoSidecar(err) {
			return
		}
		fmt.Fprintf(os.Stderr, "whitelabel: sidecar rejected: %v\n", err)
		return
	}
	settings, err := appconfig.LoadAppSettings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "whitelabel: load settings: %v\n", err)
		return
	}
	// Already pinned to this Hub → no-op.
	if settings.LockedHubURL == prof.HubURL && appconfig.AppMode(settings.AppMode) == appconfig.ModeEndUser {
		return
	}
	settings.AppMode = string(appconfig.ModeEndUser)
	settings.LockedHubURL = prof.HubURL
	settings.Reseller.HubURL = prof.HubURL
	settings.Reseller.TenantSlug = prof.TenantSlug
	// Carry brand assets through so the EndUser shell can render the
	// distributor's identity, not stock Lurus. (Audit blocker #2.)
	settings.BrandName = prof.BrandName
	settings.BrandLogoBase64 = prof.LogoBase64
	settings.BrandPrimaryColor = prof.PrimaryColor
	settings.BrandSupportContact = prof.SupportContact
	if err := appconfig.SaveAppSettings(settings); err != nil {
		fmt.Fprintf(os.Stderr, "whitelabel: save settings: %v\n", err)
		return
	}
}

// ─── Hub preflight check (audit checklist #2) ──────────────────────

// PreflightCheck is one verdict in the WhiteLabelPreflight report.
type PreflightCheck struct {
	ID       string `json:"id"`
	Pass     bool   `json:"pass"`
	TitleZh  string `json:"titleZh"`
	TitleEn  string `json:"titleEn"`
	DetailZh string `json:"detailZh,omitempty"`
	DetailEn string `json:"detailEn,omitempty"`
}

// PreflightReport bundles every check + an overall verdict so the UI
// can show a one-line summary plus expandable details.
type PreflightReport struct {
	OK     bool             `json:"ok"`
	Checks []PreflightCheck `json:"checks"`
}

// WhiteLabelPreflight pings the Reseller's Hub to confirm the endpoints
// the EndUser binary will actually call (redeem + heartbeat) before the
// Reseller wastes a build cycle on a Hub that's missing routes. Each
// check has a 5s timeout — total preflight settles in ~10s worst case.
func (a *App) WhiteLabelPreflight(hubURL, tenantSlug string) (*PreflightReport, error) {
	hubURL = strings.TrimRight(strings.TrimSpace(hubURL), "/")
	if hubURL == "" {
		return nil, errors.New("Hub URL is required")
	}
	if _, err := url.Parse(hubURL); err != nil {
		return nil, fmt.Errorf("Hub URL parse: %w", err)
	}
	op := a.activityBus.Op("", "Hub 预检", "Hub preflight")

	ctx, cancel := context.WithTimeout(a.hubCtx(), 15*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 5 * time.Second}

	report := &PreflightReport{Checks: []PreflightCheck{}}

	op.Progress("HEAD Hub 根路径", "HEAD Hub root", 25, 4, 1)
	report.Checks = append(report.Checks, headCheck(ctx, client, "hub-root", hubURL,
		"Hub 根路径可达", "Hub root reachable"))

	op.Progress("探测 redeem 端点", "Probing redeem endpoint", 50, 4, 2)
	redeemURL := hubURL + "/api/v2/switch/redeem"
	report.Checks = append(report.Checks, postCheck(ctx, client, "redeem", redeemURL,
		`{"code":"__preflight__","fingerprint":"__preflight__"}`,
		"redeem 端点存在", "Redeem endpoint exists"))

	op.Progress("探测 heartbeat 端点", "Probing heartbeat endpoint", 75, 4, 3)
	hbURL := hubURL + "/api/v2/switch/heartbeat"
	if tenantSlug != "" {
		hbURL = fmt.Sprintf("%s/api/v2/%s/user/heartbeat", hubURL, strings.TrimSpace(tenantSlug))
	}
	report.Checks = append(report.Checks, postCheck(ctx, client, "heartbeat", hbURL,
		`{}`,
		"heartbeat 端点存在", "Heartbeat endpoint exists"))

	op.Progress("Reseller 配置就绪检查", "Reseller config ready check", 90, 4, 4)
	report.Checks = append(report.Checks, resellerConfigCheck())

	report.OK = true
	for _, c := range report.Checks {
		if !c.Pass {
			report.OK = false
			break
		}
	}
	if report.OK {
		op.Done("全部通过", "All checks passed")
	} else {
		op.Error("有检查项未通过 — 见报告")
	}
	return report, nil
}

func headCheck(ctx context.Context, client *http.Client, id, url string, zh, en string) PreflightCheck {
	req, _ := http.NewRequestWithContext(ctx, "HEAD", url+"/", nil)
	resp, err := client.Do(req)
	if err != nil {
		return PreflightCheck{ID: id, Pass: false, TitleZh: zh, TitleEn: en, DetailEn: err.Error()}
	}
	defer resp.Body.Close()
	pass := resp.StatusCode < 500
	return PreflightCheck{
		ID: id, Pass: pass, TitleZh: zh, TitleEn: en,
		DetailEn: fmt.Sprintf("HTTP %d", resp.StatusCode),
	}
}

// postCheck considers the endpoint "exists" when the server returns
// anything other than 404 / network error. 4xx (auth/validation) is fine —
// it means the route is wired up; only 404 means the route is missing.
func postCheck(ctx context.Context, client *http.Client, id, url, body, zh, en string) PreflightCheck {
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		return PreflightCheck{ID: id, Pass: false, TitleZh: zh, TitleEn: en, DetailEn: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return PreflightCheck{ID: id, Pass: false, TitleZh: zh, TitleEn: en, DetailEn: err.Error()}
	}
	defer resp.Body.Close()
	pass := resp.StatusCode != http.StatusNotFound
	return PreflightCheck{
		ID: id, Pass: pass, TitleZh: zh, TitleEn: en,
		DetailZh: fmt.Sprintf("HTTP %d（404=未实现，其他都算路由存在）", resp.StatusCode),
		DetailEn: fmt.Sprintf("HTTP %d (404=missing, anything else means route is wired)", resp.StatusCode),
	}
}

func resellerConfigCheck() PreflightCheck {
	s, err := appconfig.LoadAppSettings()
	if err != nil {
		return PreflightCheck{
			ID: "reseller-cfg", Pass: false,
			TitleZh: "本机 Reseller 配置就绪", TitleEn: "Local Reseller config ready",
			DetailEn: err.Error(),
		}
	}
	pass := s.Reseller.HubURL != "" && s.Reseller.AdminToken != ""
	d := PreflightCheck{
		ID: "reseller-cfg", Pass: pass,
		TitleZh: "本机 Reseller 配置就绪", TitleEn: "Local Reseller config ready",
	}
	if !pass {
		d.DetailZh = "请先在 Reseller Setup Wizard 配置 Hub URL + Admin Token"
		d.DetailEn = "Run Reseller Setup Wizard to fill Hub URL + Admin Token first"
	}
	return d
}

// ─── Output folder + ZIP (audit polish #1) ─────────────────────────

// OpenWhiteLabelOutputDir opens the system file explorer at the build
// output directory. Wraps the existing helper so the frontend doesn't
// have to know about platform-specific shell calls.
func (a *App) OpenWhiteLabelOutputDir(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return errors.New("output directory is required")
	}
	return openDirectory(dir)
}

// ZipWhiteLabelOutputDir wraps the build result into a single .zip the
// Reseller can hand off as one file (vs the loose dir + sidecar combo).
// The zip lands next to the directory it bundles, named "<basename>.zip".
func (a *App) ZipWhiteLabelOutputDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", errors.New("output directory is required")
	}
	stat, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", dir, err)
	}
	if !stat.IsDir() {
		return "", fmt.Errorf("not a directory: %s", dir)
	}
	op := a.activityBus.Op("", "打 ZIP 包", "Zipping package")

	zipPath := strings.TrimRight(dir, string(os.PathSeparator)) + ".zip"
	f, err := os.Create(zipPath)
	if err != nil {
		op.Error(err.Error())
		return "", fmt.Errorf("create zip: %w", err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)

	walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(w, src)
		return err
	})
	if walkErr != nil {
		_ = zw.Close()
		op.Error(walkErr.Error())
		return "", fmt.Errorf("walk: %w", walkErr)
	}
	if err := zw.Close(); err != nil {
		op.Error(err.Error())
		return "", fmt.Errorf("close zip: %w", err)
	}
	op.Done("ZIP 已生成 — "+zipPath, "ZIP ready — "+zipPath)
	return zipPath, nil
}

// ─── Build history (audit polish #2) ────────────────────────────────

// BuildHistoryEntry is one row in the persisted build log.
type BuildHistoryEntry struct {
	BuiltAt    time.Time `json:"builtAt"`
	BrandName  string    `json:"brandName"`
	HubURL     string    `json:"hubUrl"`
	BinaryPath string    `json:"binaryPath"`
	SHA256     string    `json:"sha256"`
}

func buildHistoryPath() string {
	return filepath.Join(appDataBaseDir(), "whitelabel-history.jsonl")
}

func appendBuildHistory(e BuildHistoryEntry) {
	path := buildHistoryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	body, err := json.Marshal(e)
	if err != nil {
		return
	}
	_, _ = f.Write(append(body, '\n'))
}

// ListWhiteLabelBuilds returns the most-recent N build records, newest
// first. Used by PackagerPage to show a "previous builds" list so the
// Reseller can find an old binary path quickly.
func (a *App) ListWhiteLabelBuilds(max int) ([]BuildHistoryEntry, error) {
	if max <= 0 {
		max = 10
	}
	data, err := os.ReadFile(buildHistoryPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []BuildHistoryEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e BuildHistoryEntry
		if json.Unmarshal([]byte(line), &e) == nil {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].BuiltAt.After(out[j].BuiltAt) })
	if len(out) > max {
		out = out[:max]
	}
	return out, nil
}
