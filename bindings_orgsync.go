package main

import (
	"fmt"

	"lurus-switch/internal/capability"
	"lurus-switch/internal/orgsync"
)

// Org sync surface is the foundation for Enterprise mode — the
// company's department tree + employee identity records. Reads are
// open (any UI element may render the org chart); writes are
// CapUserCreate / CapUserModify gated.

// orgSyncStore lazily initializes (so a fresh Personal install doesn't
// pay the file-IO cost). Lives on services in a follow-up; for now we
// stick it on the binding file to keep the integration churn minimal.
func (a *App) orgsyncStore() (*orgsync.Store, error) {
	a.services.orgsyncMu.Lock()
	defer a.services.orgsyncMu.Unlock()
	if a.services.orgsync == nil {
		s, err := orgsync.NewStore(appDataBaseDir())
		if err != nil {
			return nil, err
		}
		a.services.orgsync = s
	}
	return a.services.orgsync, nil
}

// === Departments ===

// ListDepartments returns all departments sorted by name. Open read.
func (a *App) ListDepartments() ([]orgsync.Department, error) {
	store, err := a.orgsyncStore()
	if err != nil {
		return nil, err
	}
	return store.ListDepartments(), nil
}

// GetDepartmentTree returns the rooted hierarchy with employee counts.
// Drives the Enterprise-mode org chart panel.
func (a *App) GetDepartmentTree() ([]orgsync.TreeNode, error) {
	store, err := a.orgsyncStore()
	if err != nil {
		return nil, err
	}
	return store.Tree(), nil
}

// CreateDepartment requires CapUserCreate (department setup is an
// HR-admin / IT-admin action — the closest existing cap).
func (a *App) CreateDepartment(d orgsync.Department) (*orgsync.Department, error) {
	if err := capability.RequireCurrent(capability.CapUserCreate); err != nil {
		return nil, err
	}
	store, err := a.orgsyncStore()
	if err != nil {
		return nil, err
	}
	return store.CreateDepartment(d)
}

// UpdateDepartment patches a department.
func (a *App) UpdateDepartment(d orgsync.Department) (*orgsync.Department, error) {
	if err := capability.RequireCurrent(capability.CapUserModify); err != nil {
		return nil, err
	}
	store, err := a.orgsyncStore()
	if err != nil {
		return nil, err
	}
	return store.UpdateDepartment(d)
}

// DeleteDepartment deletes only when the department is empty.
func (a *App) DeleteDepartment(id string) error {
	if err := capability.RequireCurrent(capability.CapUserDelete); err != nil {
		return err
	}
	store, err := a.orgsyncStore()
	if err != nil {
		return err
	}
	return store.DeleteDepartment(id)
}

// === Employees ===

// ListEmployees with optional filter by department + activeOnly.
func (a *App) ListEmployees(deptID string, activeOnly bool) ([]orgsync.Employee, error) {
	store, err := a.orgsyncStore()
	if err != nil {
		return nil, err
	}
	return store.ListEmployees(deptID, activeOnly), nil
}

// CreateEmployee adds a new identity to the org. SSO subject left
// blank — populated on first SSO login via UpdateEmployee.
func (a *App) CreateEmployee(e orgsync.Employee) (*orgsync.Employee, error) {
	if err := capability.RequireCurrent(capability.CapUserCreate); err != nil {
		return nil, err
	}
	store, err := a.orgsyncStore()
	if err != nil {
		return nil, err
	}
	return store.CreateEmployee(e)
}

// UpdateEmployee patches mutable fields. SSO subject is locked once set.
func (a *App) UpdateEmployee(patch orgsync.Employee) (*orgsync.Employee, error) {
	if err := capability.RequireCurrent(capability.CapUserModify); err != nil {
		return nil, err
	}
	store, err := a.orgsyncStore()
	if err != nil {
		return nil, err
	}
	return store.UpdateEmployee(patch)
}

// DeactivateEmployee flips Active=false (preserves the record for
// audit / chargeback continuity). Use this on offboarding rather than
// hard-deleting.
func (a *App) DeactivateEmployee(id string) error {
	if err := capability.RequireCurrent(capability.CapUserFreeze); err != nil {
		return err
	}
	store, err := a.orgsyncStore()
	if err != nil {
		return err
	}
	return store.DeactivateEmployee(id)
}

// FindEmployeeByEmail is an open read used by ticket-routing and SSO
// pre-checks ("does this email exist before we create it?").
func (a *App) FindEmployeeByEmail(email string) (*orgsync.Employee, error) {
	store, err := a.orgsyncStore()
	if err != nil {
		return nil, err
	}
	e := store.FindEmployeeByEmail(email)
	if e == nil {
		return nil, fmt.Errorf("employee with email %q not found", email)
	}
	return e, nil
}
