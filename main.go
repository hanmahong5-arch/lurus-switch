package main

import (
	"context"
	"embed"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"lurus-switch/internal/bashguard"
	"lurus-switch/internal/deeplink"
	"lurus-switch/internal/diagnostics"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Stamp t0 for the startup-performance trace before anything else runs.
	diagnostics.Default.MarkStart()

	// CLI fast-path: when invoked as the Bash-Guard hook (Claude Code's
	// PreToolUse), read JSON from stdin, evaluate, and exit BEFORE the
	// Wails GUI initialises. This is what lets the same lurus-switch.exe
	// double as the hook executable so the user doesn't need a separate
	// binary on PATH.
	if len(os.Args) > 1 && os.Args[1] == "--bashguard" {
		eng, err := bashguard.NewEngine(bashguard.DefaultRules())
		if err != nil {
			os.Stderr.WriteString("[lurus-bashguard] init: " + err.Error() + "\n")
			os.Exit(0) // fail-open
		}
		logPath := filepath.Join(appDataBaseDir(), "bashguard-blocks.jsonl")
		os.Exit(bashguard.HandleStdin(os.Stdin, os.Stderr, logPath, eng))
	}

	app := NewApp()
	diagnostics.Default.Mark("services-init")

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
			// startup() returning means the GUI is interactive — background
			// services may still be settling on safeGo goroutines.
			diagnostics.Default.MarkGUIReady()
			if dlServer != nil {
				dlServer.Start(ctx, deeplink.MakeWailsHandler(ctx))
			}
			if startupURL != "" {
				if payload, perr := deeplink.Parse(startupURL); perr == nil {
					deeplink.MakeWailsHandler(ctx)(payload)
				}
			}
			// Persist the trace after the window is up so the next launch
			// can show a delta. Best-effort — never block startup on disk IO.
			go safeGo("startup-trace-persist", func() {
				_, _ = diagnostics.Default.Persist(appDataBaseDir())
			})
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
