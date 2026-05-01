// Package redemption implements the EndUser activation lifecycle:
//
//   - DeviceFingerprint:  a stable, machine-bound identifier sent with the
//     redemption request and persisted alongside the resulting token.
//   - Store:              encrypted on-disk persistence of the activation
//     payload (user token, quota, expiry, fingerprint).
//   - Redeem:              the HTTP client that exchanges a redemption code
//     for a Hub-issued user token.
//   - Heartbeat:           periodic liveness probe so a revoked token gets
//     evicted within minutes instead of waiting for the next quota call.
//
// Threat model: the encryption key is derived from machine metadata
// (hostname + username + first non-loopback MAC) via SHA-256. This makes
// the activation file useless when copied to a different machine — but it
// is *not* a defense against a privileged attacker on the same machine,
// who could re-derive the key trivially. That tradeoff is intentional:
// EndUser builds run as the logged-in desktop user with no elevated
// secrets, so binding to host identity is the right cost/value point.
package redemption

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/user"
	"runtime"
	"sort"
	"strings"
)

// fingerprintLength is the truncated hex length surfaced to UIs and the
// Hub. 16 chars (64 bits) is more than enough collision resistance for the
// "lock token to one device" use case while staying short enough to log
// without wrapping.
const fingerprintLength = 16

// DeviceFingerprint returns a stable identifier for the current host.
// The same machine returns the same value across runs and OS upgrades;
// a different machine returns a different value with overwhelming probability.
//
// Inputs (all best-effort — missing inputs are skipped, never fatal):
//   - hostname (os.Hostname)
//   - desktop username (user.Current)
//   - GOOS / GOARCH (so a dual-boot Windows/Linux on the same iron is treated
//     as two separate devices, matching how the user actually sees them)
//   - sorted, comma-joined list of non-loopback hardware MAC addresses
//
// MAC enumeration uses the first valid permanent interface set; sorting
// keeps the output stable across reboots when interface ordering shifts
// (common on laptops with virtual VPN adapters).
func DeviceFingerprint() string {
	parts := make([]string, 0, 5)
	parts = append(parts, "lurus-switch:enduser")
	parts = append(parts, runtime.GOOS, runtime.GOARCH)

	if h, err := os.Hostname(); err == nil && h != "" {
		parts = append(parts, "host:"+h)
	}
	if u, err := user.Current(); err == nil && u != nil && u.Username != "" {
		parts = append(parts, "user:"+u.Username)
	}
	if macs := collectStableMACs(); len(macs) > 0 {
		parts = append(parts, "mac:"+strings.Join(macs, ","))
	}

	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])[:fingerprintLength]
}

// collectStableMACs returns the lowercase hex MACs of all non-loopback,
// non-virtual hardware interfaces, sorted lexicographically.
//
// "non-virtual" is approximated by HardwareAddr length == 6 and FlagUp is
// not required (a disabled NIC still identifies the device). We exclude
// docker / vmnet style addresses by skipping interfaces whose names start
// with the well-known prefixes — imperfect but good enough to absorb the
// common churn cases.
func collectStableMACs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	skipPrefix := []string{"docker", "veth", "br-", "vmnet", "vboxnet", "lo"}
	out := make([]string, 0, len(ifaces))
	for _, ifc := range ifaces {
		if len(ifc.HardwareAddr) != 6 {
			continue
		}
		if ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := strings.ToLower(ifc.Name)
		skip := false
		for _, p := range skipPrefix {
			if strings.HasPrefix(name, p) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		out = append(out, strings.ToLower(ifc.HardwareAddr.String()))
	}
	sort.Strings(out)
	return out
}

// fingerprintKey derives a 32-byte AES-256 key from the device fingerprint
// plus a fixed package salt. The same machine always derives the same key,
// so an activation file written on one boot decrypts on the next without
// needing to round-trip a separate key.
func fingerprintKey() []byte {
	material := fmt.Sprintf("lurus-switch:redemption:v1:%s", DeviceFingerprint())
	sum := sha256.Sum256([]byte(material))
	return sum[:]
}
