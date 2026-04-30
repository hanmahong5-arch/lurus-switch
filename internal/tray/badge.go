// Package tray manages the system tray icon, menu, and badge for Lurus Switch.
// It uses github.com/energye/systray (a maintained fork of getlantern/systray).
//
// NOTE: Icon switching by color is implemented via tooltip text changes only.
// Generating separate colored ICO assets at runtime is not yet supported;
// multi-color .ico asset injection is left for a future milestone.
// The badge tier is reflected in the tooltip string (e.g., "[⚠] Lurus Switch").
package tray

import "fmt"

// BadgeTier represents the quota-usage severity level for the tray badge.
type BadgeTier int

const (
	// TierUnknown is used when quota data is unavailable or the gateway is down.
	TierUnknown BadgeTier = iota
	// TierGreen indicates usage ≤ 50%.
	TierGreen
	// TierYellow indicates usage between 50% and 80%.
	TierYellow
	// TierRed indicates usage > 80%.
	TierRed
	// TierGray indicates the gateway is not running (overrides quota tier).
	TierGray
)

// classifyQuota maps a UsedPercent value to a BadgeTier.
// Negative UsedPercent means unknown.
func classifyQuota(usedPercent float64) BadgeTier {
	switch {
	case usedPercent < 0:
		return TierUnknown
	case usedPercent <= 50:
		return TierGreen
	case usedPercent <= 80:
		return TierYellow
	default:
		return TierRed
	}
}

// resolveTier derives the final display tier given quota and gateway status.
// A stopped gateway always returns TierGray regardless of quota.
func resolveTier(q QuotaSnapshot, gw GatewayStatus) BadgeTier {
	if !gw.Running {
		return TierGray
	}
	return classifyQuota(q.UsedPercent)
}

// buildTooltip constructs the tray tooltip string from current state.
func buildTooltip(q QuotaSnapshot, gw GatewayStatus) string {
	tierPrefix := tierEmoji(resolveTier(q, gw))
	gwInfo := ""
	if gw.Running {
		gwInfo = fmt.Sprintf(" | Gateway :%d", gw.Port)
	} else {
		gwInfo = " | Gateway stopped"
	}
	balance := ""
	if q.BalanceText != "" {
		balance = " | " + q.BalanceText
	}
	return fmt.Sprintf("%s Lurus Switch%s%s", tierPrefix, gwInfo, balance)
}

// tierEmoji returns a short ASCII/emoji prefix for the tier (tooltip use only).
func tierEmoji(t BadgeTier) string {
	switch t {
	case TierGreen:
		return "[●]"
	case TierYellow:
		return "[◑]"
	case TierRed:
		return "[⚠]"
	case TierGray:
		return "[○]"
	default:
		return "[?]"
	}
}
