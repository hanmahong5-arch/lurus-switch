package main

import (
	"fmt"

	"lurus-switch/internal/budget"
)

// ============================
// Budget Wall Bindings
// ============================
//
// Active spend-wall controls (Wave2 BUD). The guard is wired into the
// gateway in newServices(), so configuration changes here take effect on
// the next request without restarting the gateway.

// BudgetGetConfig returns the current spend-wall configuration.
func (a *App) BudgetGetConfig() budget.Config {
	if a.budgetGuard == nil {
		return budget.DefaultConfig()
	}
	return a.budgetGuard.GetConfig()
}

// BudgetSetConfig persists new limits and applies them immediately.
func (a *App) BudgetSetConfig(c budget.Config) error {
	if a.budgetGuard == nil {
		return fmt.Errorf("budget guard not initialised — gateway disabled?")
	}
	return a.budgetGuard.SetConfig(c)
}

// BudgetGetStatus returns current usage vs. limit for the UI gauge.
func (a *App) BudgetGetStatus() budget.Status {
	if a.budgetGuard == nil {
		return budget.Status{}
	}
	return a.budgetGuard.Status()
}

// BudgetResetSession zeroes the in-process session counter so the user
// can keep working after intentionally hitting the cap.
func (a *App) BudgetResetSession() {
	if a.budgetGuard == nil {
		return
	}
	a.budgetGuard.ResetSession()
}
