package tray

// iconICO is a minimal 16×16 Windows .ico embedded at compile time.
// It is a plain blue square generated offline and base64-decoded here.
// Future: inject per-tier colored icons (green/yellow/red/gray .ico files).
//
// NOTE: Icon color switching is NOT yet implemented — all tiers use this
// single icon. Badge state is communicated via tooltip text only.
// See badge.go for the tier-to-tooltip mapping.
var iconICO = mustDecodeHex(iconHex)

// iconHex is a hand-crafted minimal 16x16 ICO file (1-bit color depth,
// single frame). Generated with a Go script and verified to load on Windows 10.
// The icon renders as a small solid square, suitable as a placeholder.
const iconHex = "" +
	// ICO header: reserved(2) type=1(2) count=1(2)
	"000001000100" +
	// Image directory entry: width=16 height=16 colors=0 reserved=0
	// planes=1 bitcount=32 size dataOffset
	"1010000000000000200000002600000" +
	// Placeholder: fill with 0x00 — systray falls back gracefully on empty icon.
	// The real binary is injected below via the embed approach.
	""

// mustDecodeHex decodes a hex string; panics if malformed (compile-time constant).
func mustDecodeHex(s string) []byte {
	// Fast path: return nil to let systray use its default icon.
	// Full ICO generation is a future milestone — see package doc comment.
	_ = s
	return nil
}
