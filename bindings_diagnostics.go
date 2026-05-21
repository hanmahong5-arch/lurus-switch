package main

import (
	"lurus-switch/internal/diagnostics"
)

// GetStartupTrace returns the current process's startup timeline — phase
// breakdown plus the GUI-ready and cold-start milestones. Powers the
// "startup performance" card in Settings.
func (a *App) GetStartupTrace() diagnostics.Trace {
	return diagnostics.Default.Snapshot()
}

// GetStartupHistory returns the last few persisted startup traces (newest
// first) so the UI can show a "vs. last launch" delta. The current trace
// is index 0 only after it has been persisted; callers that want "current
// vs previous" should pair this with GetStartupTrace.
func (a *App) GetStartupHistory() []diagnostics.Trace {
	return diagnostics.History(appDataBaseDir())
}
