package main

import (
	"fmt"

	"lurus-switch/internal/agenttemplate"
)

// Agent template gallery surface. Templates are read-only, declarative
// recipes (sales/support/ops/finance/compliance) — listing them needs
// no capability gate. Instantiation flows through the existing agent
// store and IS gated by CapAgentCreate (added in a follow-up sprint).

// ListBuiltinTemplates returns the curated agent recipe set bundled
// with this build. Order is the recommended deployment order (sales
// first, then support, ops, finance, compliance).
func (a *App) ListBuiltinTemplates() []agenttemplate.Template {
	return agenttemplate.AllTemplates()
}

// GetBuiltinTemplate returns a single template by ID. Returns an
// error when the ID is unknown — the frontend can show "template
// removed in this build" rather than a blank panel.
func (a *App) GetBuiltinTemplate(id string) (*agenttemplate.Template, error) {
	t := agenttemplate.Get(id)
	if t == nil {
		return nil, fmt.Errorf("agent template %q not found", id)
	}
	return t, nil
}
