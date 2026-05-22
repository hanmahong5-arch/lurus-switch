package main

import (
	"fmt"
	"time"

	"lurus-switch/internal/appreg"
	"lurus-switch/internal/capability"
	"lurus-switch/internal/metering"
	"lurus-switch/internal/orgsync"
)

// Chargeback dashboard bindings. Two read endpoints:
//
//   - GetChargebackReport(fromMs, toMs) — joins metering aggregations
//     with the org chart so the UI can render department / employee
//     names instead of raw IDs.
//   - SetAppOwnership(appId, employeeId, costCenter) — writes the
//     binding the gateway middleware reads on every request to
//     attribute traffic.
//
// Personal / Reseller installs leave employee + cost-center empty;
// the report still runs but every record falls into the "unattributed"
// bucket. That's fine — the chargeback page is gated to Enterprise
// mode in the sidebar.

// ChargebackRow rolls up either a department or an employee bucket
// for the dashboard table. The frontend renders the same shape twice
// (once per tab). EmployeeID is empty on department rows, DeptID empty
// on employee rows.
type ChargebackRow struct {
	Kind       string `json:"kind"` // "department" | "employee"
	DeptID     string `json:"deptId,omitempty"`
	DeptName   string `json:"deptName,omitempty"`
	EmployeeID string `json:"employeeId,omitempty"`
	Email      string `json:"email,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	CostCenter string `json:"costCenter,omitempty"`
	TotalCalls int64  `json:"totalCalls"`
	TokensIn   int64  `json:"tokensIn"`
	TokensOut  int64  `json:"tokensOut"`
	UniqueEmps int    `json:"uniqueEmployees,omitempty"` // department only
}

// ChargebackReport is the round-trip wrapper. Range is echoed back so
// the UI can confirm what was queried (avoids "did my date picker get
// applied?" ambiguity).
type ChargebackReport struct {
	FromMs       int64           `json:"fromMs"`
	ToMs         int64           `json:"toMs"`
	ByDepartment []ChargebackRow `json:"byDepartment"`
	ByEmployee   []ChargebackRow `json:"byEmployee"`
}

// GetChargebackReport returns rolled-up usage for the given time range.
// Reads are open — chargeback is a transparency surface, not a write.
// Times are unix milliseconds (matches the date picker).
func (a *App) GetChargebackReport(fromMs, toMs int64) (*ChargebackReport, error) {
	if a.meterStore == nil {
		return nil, fmt.Errorf("metering store unavailable")
	}
	if toMs == 0 {
		toMs = time.Now().UnixMilli()
	}
	from := time.UnixMilli(fromMs)
	to := time.UnixMilli(toMs)
	if !from.Before(to) {
		return nil, fmt.Errorf("invalid range: from must be before to")
	}

	// Deparment names + cost-centers come from orgsync. The store is
	// lazy in Enterprise mode, so this populates it on first call.
	store, _ := a.orgsyncStoreSafe() // best-effort — may be nil in Personal/Reseller
	deptByCC := map[string]*orgsync.Department{}
	deptByID := map[string]*orgsync.Department{}
	if store != nil {
		for _, d := range store.ListDepartments() {
			cp := d
			deptByID[d.ID] = &cp
			if d.CostCenter != "" {
				deptByCC[d.CostCenter] = &cp
			}
		}
	}
	empByID := map[string]*orgsync.Employee{}
	if store != nil {
		for _, e := range store.ListEmployees("", false) {
			cp := e
			empByID[e.ID] = &cp
		}
	}

	report := &ChargebackReport{FromMs: fromMs, ToMs: toMs}

	for _, cs := range a.meterStore.CostCenterSummaries(from, to) {
		row := ChargebackRow{
			Kind:       "department",
			CostCenter: cs.CostCenter,
			TotalCalls: cs.TotalCalls,
			TokensIn:   cs.TokensIn,
			TokensOut:  cs.TokensOut,
			UniqueEmps: cs.UniqueEmps,
		}
		if d := deptByCC[cs.CostCenter]; d != nil {
			row.DeptID = d.ID
			row.DeptName = d.Name
		} else if cs.CostCenter == "" {
			row.DeptName = "(unattributed)"
		} else {
			row.DeptName = cs.CostCenter // fall back to raw cost-center label
		}
		report.ByDepartment = append(report.ByDepartment, row)
	}

	for _, es := range a.meterStore.EmployeeSummaries(from, to) {
		row := ChargebackRow{
			Kind:       "employee",
			EmployeeID: es.EmployeeID,
			CostCenter: es.CostCenter,
			TotalCalls: es.TotalCalls,
			TokensIn:   es.TokensIn,
			TokensOut:  es.TokensOut,
		}
		if e := empByID[es.EmployeeID]; e != nil {
			row.Email = e.Email
			row.DisplayName = e.DisplayName
		} else if es.EmployeeID == "" {
			row.DisplayName = "(unattributed)"
		}
		if d := deptByCC[es.CostCenter]; d != nil {
			row.DeptID = d.ID
			row.DeptName = d.Name
		}
		report.ByEmployee = append(report.ByEmployee, row)
	}

	return report, nil
}

// SetAppOwnership binds an app to an employee + cost-center for
// chargeback. Capability-gated since granting attribution lets the
// admin redirect cost across departments. The gateway middleware
// reads the binding on every request, so changes apply immediately.
func (a *App) SetAppOwnership(appID, employeeID, costCenter string) (*appreg.App, error) {
	if err := capability.RequireCurrent(capability.CapUserModify); err != nil {
		return nil, err
	}
	if a.appRegistry == nil {
		return nil, fmt.Errorf("app registry unavailable")
	}
	return a.appRegistry.SetOwnership(appID, employeeID, costCenter)
}

// orgsyncStoreSafe returns the orgsync store without surfacing an
// error — used by chargeback because Personal / Reseller installs
// don't have an org chart and that's fine.
func (a *App) orgsyncStoreSafe() (*orgsync.Store, error) {
	store, err := a.orgsyncStore()
	if err != nil {
		return nil, err
	}
	return store, nil
}

// Suppress unused-package lint when metering import is only via type
// references in this file.
var _ = metering.Record{}
