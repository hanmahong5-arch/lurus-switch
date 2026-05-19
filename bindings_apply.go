package main

import (
	"fmt"
	"sync"

	"lurus-switch/internal/configapply"
)

// Wails bindings for the configapply package (dry-run preview + atomic apply).
//
// Design: the applier + store + registry are lazily initialized by sync.Once so
// this file is self-contained and does not require wiring through services.go.
// Subagents extending the call-site coverage should:
//   1. Register additional planners in initApplyInfrastructure() below
//   2. Wire startup PendingTransaction recovery via the app lifecycle
//   3. Replace the lazy init with a services field once the pattern proves out

var (
	applyOnce        sync.Once
	applyApplier     *configapply.Applier
	applyRegistry    *configapply.Registry
	applyInitErr     error
	applyInitMessage string
)

func initApplyInfrastructure() {
	store, err := configapply.NewStore()
	if err != nil {
		applyInitErr = fmt.Errorf("configapply: new store: %w", err)
		applyInitMessage = "Failed to initialise the configapply store directory. " +
			"Subsequent BuildChangePlan / ApplyChangePlan calls will fail-soft."
		return
	}
	applyApplier = configapply.NewApplier(store)
	applyRegistry = configapply.NewRegistry()

	// Baseline planner: single-file write. Sub-agents register more granular
	// planners (save-claude-config, deeplink-import, ...) when they wire call sites.
	if err := applyRegistry.Register(configapply.SaveSingleFilePlanner{
		IntentName: "save-single-file",
		DescribeFn: func(p map[string]any) string {
			path, _ := p["path"].(string)
			if path == "" {
				return "写入文件(路径未指定)"
			}
			return fmt.Sprintf("写入文件 %s", path)
		},
	}); err != nil {
		applyInitErr = fmt.Errorf("configapply: register baseline planner: %w", err)
		applyInitMessage = "Baseline 'save-single-file' planner registration failed."
		return
	}
}

func ensureApply() error {
	applyOnce.Do(initApplyInfrastructure)
	return applyInitErr
}

// BuildChangePlan generates a dry-run preview for the given intent. The
// returned ChangePlan carries before/after content, a unified diff and a
// per-file +N -M summary so the frontend can render a Monaco diff modal
// before the user confirms.
//
// intent must be a key registered with the planner registry. params semantics
// are intent-specific; for 'save-single-file' provide 'path' and 'content'
// strings.
//
// Returns the plan and a Go error only when the intent is unknown or planner
// fails outright. Per the no-silent-failure design, this is the only case in
// the apply lifecycle where a Go error escapes.
func (a *App) BuildChangePlan(intent string, params map[string]interface{}) (*configapply.ChangePlan, error) {
	if err := ensureApply(); err != nil {
		return nil, err
	}
	return applyRegistry.Plan(intent, params)
}

// ApplyChangePlan applies a previously-built plan and returns a self-describing
// ApplyResult. Always returns a non-nil result; the caller must inspect
// .Success rather than relying on error-return semantics (see
// memory/feedback_wails_result_success.md for the failure pattern this guards
// against).
func (a *App) ApplyChangePlan(plan configapply.ChangePlan) configapply.ApplyResult {
	if err := ensureApply(); err != nil {
		return configapply.ApplyResult{
			PlanID:       plan.ID,
			Success:      false,
			Phase:        configapply.PhasePending,
			WhatHappened: applyInitMessage,
			WhatExpected: "configapply 应在 Switch 启动后准备就绪。",
			RawError:     err.Error(),
			NextSteps: []configapply.NextStep{
				{Label: "查看详细日志", Action: "open_log"},
				{Label: "重启 Switch", Action: "restart_app"},
			},
		}
	}
	result := applyApplier.Apply(&plan)
	if result == nil {
		return configapply.ApplyResult{
			PlanID:       plan.ID,
			Success:      false,
			Phase:        configapply.PhasePending,
			WhatHappened: "applier 返回了空结果,这不应该发生。",
			WhatExpected: "Apply 应总是返回非空 ApplyResult。",
			RawError:     "nil result from applier",
		}
	}
	return *result
}

// ListApplyIntents exposes the registered planner intents to the frontend so
// debug surfaces (Settings → Diagnostics) can verify what mutation paths are
// wired through configapply.
func (a *App) ListApplyIntents() []string {
	if err := ensureApply(); err != nil {
		return nil
	}
	return applyRegistry.Intents()
}
