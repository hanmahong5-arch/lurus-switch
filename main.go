package main

import (
	"context"
	"embed"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"lurus-switch/internal/deeplink"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	// Deep-link single-instance guard.
	// If another Switch process already holds the IPC channel, forward our
	// startup URL (if any) to it and exit — the running instance will handle
	// the import dialog. Falls back silently when IPC init fails so a broken
	// IPC channel never blocks the user from launching the app.
	dataDir := appDataBaseDir()
	dlServer, dlErr := deeplink.NewServer(dataDir)
	if dlErr == deeplink.ErrAlreadyRunning {
		if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], deeplink.Scheme+"://") {
			_ = deeplink.SendToExisting(dataDir, os.Args[1])
		}
		return
	}
	if dlErr != nil {
		println("Warning: deeplink server init failed:", dlErr.Error())
		dlServer = nil
	}

	// Register the OS protocol handler (HKCU-only on Windows; stub on macOS/Linux).
	if exePath, exeErr := os.Executable(); exeErr == nil {
		_ = deeplink.Register(exePath)
	}

	// Capture a deep-link URL passed as argv[1] (OS handler launch).
	var startupURL string
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], deeplink.Scheme+"://") {
		startupURL = os.Args[1]
	}

	err := wails.Run(&options.App{
		Title:  "lurus-switch",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			if dlServer != nil {
				dlServer.Start(ctx, deeplink.MakeWailsHandler(ctx))
			}
			if startupURL != "" {
				if payload, perr := deeplink.Parse(startupURL); perr == nil {
					deeplink.MakeWailsHandler(ctx)(payload)
				}
			}
		},
		OnShutdown: func(ctx context.Context) {
			app.shutdown(ctx)
			if dlServer != nil {
				_ = dlServer.Stop()
			}
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
