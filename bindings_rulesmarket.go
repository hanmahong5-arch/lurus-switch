package main

import (
	"context"

	"lurus-switch/internal/rulesmarket"
)

// RulesMarketResult is the {success, message} envelope returned by mutation
// bindings.  Wails generates a TS interface from this struct.
type RulesMarketResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RulesMarketWriteResult mirrors rulesmarket.WriteResult plus the standard
// {success, message} envelope so the frontend can handle errors with the same
// pattern used elsewhere (result.success guard).
type RulesMarketWriteResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
	Appended bool   `json:"appended,omitempty"`
	Skipped  bool   `json:"skipped,omitempty"`
}

// RulesMarketList returns the merged builtin + cached remote rule templates.
func (a *App) RulesMarketList() ([]rulesmarket.RuleTemplate, error) {
	return rulesmarket.NewMarket().ListTemplates()
}

// RulesMarketRefresh fetches a remote manifest at url and stores it in the
// local ETag-keyed cache.  Empty url is rejected.
func (a *App) RulesMarketRefresh(url string) RulesMarketResult {
	if url == "" {
		return RulesMarketResult{Success: false, Message: "url required"}
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if err := rulesmarket.NewMarket().RefreshFromRemote(ctx, url); err != nil {
		return RulesMarketResult{Success: false, Message: err.Error()}
	}
	return RulesMarketResult{Success: true, Message: "ok"}
}

// RulesMarketWrite installs a rule template into the project directory in the
// requested target format.  When overwrite is false and the target file
// already contains the template, the call is a no-op (Skipped=true).
func (a *App) RulesMarketWrite(projectDir string, tmpl rulesmarket.RuleTemplate, format string, overwrite bool) RulesMarketWriteResult {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	res, err := rulesmarket.NewMarket().WriteRuleToProject(ctx, projectDir, tmpl, rulesmarket.Format(format), overwrite)
	if err != nil {
		return RulesMarketWriteResult{Success: false, Message: err.Error()}
	}
	msg := "written"
	if res.Appended {
		msg = "appended"
	} else if res.Skipped {
		msg = "already present"
	}
	return RulesMarketWriteResult{
		Success:  true,
		Message:  msg,
		Path:     res.Path,
		Appended: res.Appended,
		Skipped:  res.Skipped,
	}
}
