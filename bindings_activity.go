package main

import (
	"errors"
	"strings"
)

// ============================
// Activity / Frontend Telemetry Bindings
// ============================
//
// Frontend-emitted breadcrumbs that the Go side persists so users can
// hand a useful "what crashed" payload to support without needing a
// devtools console open at the moment of the failure.

// LogFrontendError persists a React ErrorBoundary catch to the audit
// journal. Called from the ErrorBoundary's componentDidCatch so a
// hard-to-reproduce render crash leaves a durable trace alongside the
// rest of Switch's state-mutation log. Returns an error only if the
// payload is unusable; the journal write itself is best-effort.
//
// boundary identifies which boundary caught it (e.g. "page:settings"),
// message + stack come straight from the React Error/ErrorInfo, and
// page is the activeTool slug so support can correlate against the
// route the user was on.
func (a *App) LogFrontendError(boundary, message, stack, page string) error {
	if a.auditJournal == nil {
		return errors.New("audit journal not initialised")
	}
	boundary = strings.TrimSpace(boundary)
	if boundary == "" {
		boundary = "unknown"
	}
	payload := map[string]any{
		"boundary": boundary,
		"page":     page,
		"message":  message,
		"stack":    stack,
	}
	a.auditJournal.RecordSystem("frontend", "frontend_error", boundary, nil, payload, nil)
	return nil
}
