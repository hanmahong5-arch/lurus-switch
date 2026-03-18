package main

import (
	"context"
	"fmt"
	"os"

	"lurus-switch/internal/gy"
	"lurus-switch/internal/toolmanifest"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ============================
// GY Product Suite Methods
// ============================

// GetGYProducts returns the built-in GY suite product definitions.
func (a *App) GetGYProducts() []gy.GYProduct {
	return gy.BuiltinProducts()
}

// CheckGYStatus runs concurrent availability checks for all GY products.
func (a *App) CheckGYStatus() []gy.GYStatus {
	products := gy.BuiltinProducts()
	ctx, cancel := context.WithTimeout(a.ctx, 6*1e9) // 6s
	defer cancel()
	return gy.CheckStatus(ctx, products)
}

// LaunchGYProduct opens the specified GY product.
// For web/service products this opens the default browser.
// For desktop products this starts the local executable.
func (a *App) LaunchGYProduct(productID string) error {
	var target *gy.GYProduct
	for _, p := range gy.BuiltinProducts() {
		if p.ID == productID {
			cp := p
			target = &cp
			break
		}
	}
	if target == nil {
		return fmt.Errorf("unknown GY product: %q", productID)
	}

	// Retrieve user token for SSO link (best-effort)
	userToken := ""
	if a.proxyMgr != nil {
		userToken = a.proxyMgr.GetSettings().UserToken
	}

	return gy.Launch(*target, func(url string) {
		wailsRuntime.BrowserOpenURL(a.ctx, url)
	}, userToken)
}

// DownloadCreator downloads the Lurus Creator installer for the current platform
// and launches it automatically. Progress is emitted as "gy:creator:progress" {percent: N}.
func (a *App) DownloadCreator() error {
	// Validate platform before starting the download.
	if ok, reason := toolmanifest.IsSupportedPlatform(); !ok {
		return fmt.Errorf("%s", reason)
	}
	return gy.DownloadCreator(a.ctx, a.loadManifest(), os.TempDir(), func(pct int) {
		wailsRuntime.EventsEmit(a.ctx, "gy:creator:progress", map[string]any{"percent": pct})
	})
}

