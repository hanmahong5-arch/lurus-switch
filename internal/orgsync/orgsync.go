// Package orgsync models the company org chart for Switch's Enterprise
// mode. Two top-level entities:
//
//   - Department: tree node. Has parent_id (root departments have ""),
//     a cost_center identifier (used to roll up token usage for finance
//     chargeback), and a display name. Tree depth and shape are
//     unconstrained — companies range from flat to deeply nested.
//
//   - Employee: leaf identity. Bound to exactly one Department. Carries
//     SSO subject (the OIDC `sub` claim once SSO is wired), email,
//     display name, role within Switch, and an optional manager_id
//     pointer for approval-routing workflows.
//
// Storage is a single JSON file at $APPDATA/lurus-switch/orgsync.json.
// In production this will sync from SCIM / Workday / LDAP via a separate
// connector package; for now everything is manual via the Wails admin
// surface.
package orgsync

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	storeFileName = "orgsync.json"
	rootParentID  = "" // root departments have no parent
)

// Role identifies an employee's authority within Switch. Maps loosely
// onto capability bundles defined in internal/capability.
type Role string

const (
	RoleEmployee   Role = "employee"   // self-service only
	RoleTeamLead   Role = "team_lead"  // sees own team
	RoleDeptAdmin  Role = "dept_admin" // manages dept config + budgets
	RoleITAdmin    Role = "it_admin"   // full Switch admin
	RoleCompliance Role = "compliance" // read-only audit access
	RoleFinance    Role = "finance"    // read chargeback reports
)

// Department is a node in the org tree.
type Department struct {
	ID         string    `json:"id"`
	ParentID   string    `json:"parentId"`   // "" for root
	Name       string    `json:"name"`
	CostCenter string    `json:"costCenter"` // matches metering.Record.CostCenter
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// Employee binds an SSO identity to a Department + Role.
type Employee struct {
	ID           string    `json:"id"`
	SSOSubject   string    `json:"ssoSubject,omitempty"` // OIDC sub claim
	Email        string    `json:"email"`
	DisplayName  string    `json:"displayName"`
	DepartmentID string    `json:"departmentId"`
	Role         Role      `json:"role"`
	ManagerID    string    `json:"managerId,omitempty"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Store persists departments + employees + provides traversal.
type Store struct {
	mu          sync.RWMutex
	path        string
	Departments map[string]*Department `json:"departments"`
	Employees   map[string]*Employee   `json:"employees"`
	idCounter   uint64                 // monotonic, used in NewID
}

// NewStore opens the store rooted at appDataDir/orgsync.json. Creates
// the file with an empty schema if it doesn't exist.
func NewStore(appDataDir string) (*Store, error) {
	if err := os.MkdirAll(appDataDir, 0o755); err != nil {
		return nil, fmt.Errorf("orgsync: ensure app dir: %w", err)
	}
	s := &Store{
		path:        filepath.Join(appDataDir, storeFileName),
		Departments: make(map[string]*Department),
		Employees:   make(map[string]*Employee),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil // empty store; we'll persist on first write
	}
	if err != nil {
		return fmt.Errorf("orgsync: read: %w", err)
	}
	var on struct {
		Departments map[string]*Department `json:"departments"`
		Employees   map[string]*Employee   `json:"employees"`
		Counter     uint64                 `json:"counter"`
	}
	if err := json.Unmarshal(data, &on); err != nil {
		return fmt.Errorf("orgsync: decode: %w", err)
	}
	s.mu.Lock()
	if on.Departments != nil {
		s.Departments = on.Departments
	}
	if on.Employees != nil {
		s.Employees = on.Employees
	}
	s.idCounter = on.Counter
	s.mu.Unlock()
	return nil
}

func (s *Store) saveLocked() error {
	on := struct {
		Departments map[string]*Department `json:"departments"`
		Employees   map[string]*Employee   `json:"employees"`
		Counter     uint64                 `json:"counter"`
	}{
		Departments: s.Departments,
		Employees:   s.Employees,
		Counter:     s.idCounter,
	}
	data, err := json.MarshalIndent(on, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

// --- Department ops -----------------------------------------------------

// CreateDepartment validates and stores a new department. Auto-generates
// the ID; CreatedAt/UpdatedAt are set.
func (s *Store) CreateDepartment(d Department) (*Department, error) {
	if strings.TrimSpace(d.Name) == "" {
		return nil, fmt.Errorf("department name required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if d.ParentID != rootParentID {
		if _, ok := s.Departments[d.ParentID]; !ok {
			return nil, fmt.Errorf("parent department %q not found", d.ParentID)
		}
	}
	d.ID = s.nextID("dept")
	d.CreatedAt = time.Now()
	d.UpdatedAt = d.CreatedAt
	if d.CostCenter == "" {
		// Default cost-center mirrors the dept ID — operators can edit later.
		d.CostCenter = d.ID
	}
	s.Departments[d.ID] = &d
	if err := s.saveLocked(); err != nil {
		delete(s.Departments, d.ID)
		return nil, err
	}
	return &d, nil
}

// UpdateDepartment patches name / costCenter / parent. ID is immutable.
func (s *Store) UpdateDepartment(d Department) (*Department, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.Departments[d.ID]
	if !ok {
		return nil, fmt.Errorf("department %q not found", d.ID)
	}
	if strings.TrimSpace(d.Name) != "" {
		existing.Name = d.Name
	}
	if strings.TrimSpace(d.CostCenter) != "" {
		existing.CostCenter = d.CostCenter
	}
	if d.ParentID != existing.ParentID {
		// Prevent cycles: walk new parent's chain to root, fail if d.ID appears.
		if d.ParentID != rootParentID {
			cursor := d.ParentID
			for cursor != rootParentID {
				if cursor == d.ID {
					return nil, fmt.Errorf("cycle detected — cannot reparent %q under itself", d.ID)
				}
				next, ok := s.Departments[cursor]
				if !ok {
					return nil, fmt.Errorf("ancestor %q not found", cursor)
				}
				cursor = next.ParentID
			}
		}
		existing.ParentID = d.ParentID
	}
	existing.UpdatedAt = time.Now()
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return existing, nil
}

// DeleteDepartment removes a department only if it has no children
// (sub-departments or employees). Caller must reassign first.
func (s *Store) DeleteDepartment(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Departments[id]; !ok {
		return fmt.Errorf("department %q not found", id)
	}
	for _, d := range s.Departments {
		if d.ParentID == id {
			return fmt.Errorf("department %q has child department %q — reassign or delete children first", id, d.ID)
		}
	}
	for _, e := range s.Employees {
		if e.DepartmentID == id {
			return fmt.Errorf("department %q has employee %q — reassign first", id, e.ID)
		}
	}
	delete(s.Departments, id)
	return s.saveLocked()
}

// ListDepartments returns all departments sorted by name.
func (s *Store) ListDepartments() []Department {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Department, 0, len(s.Departments))
	for _, d := range s.Departments {
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// DepartmentTree returns root-down hierarchy. Each node carries Children IDs.
type TreeNode struct {
	Department Department `json:"department"`
	Children   []TreeNode `json:"children"`
	EmployeeCount int     `json:"employeeCount"`
}

// Tree builds the org tree from current state. Multiple roots allowed
// (companies with disconnected divisions).
func (s *Store) Tree() []TreeNode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	childrenByParent := map[string][]Department{}
	for _, d := range s.Departments {
		childrenByParent[d.ParentID] = append(childrenByParent[d.ParentID], *d)
	}
	empCount := map[string]int{}
	for _, e := range s.Employees {
		empCount[e.DepartmentID]++
	}
	var build func(parent string) []TreeNode
	build = func(parent string) []TreeNode {
		kids := childrenByParent[parent]
		sort.Slice(kids, func(i, j int) bool { return kids[i].Name < kids[j].Name })
		out := make([]TreeNode, 0, len(kids))
		for _, d := range kids {
			out = append(out, TreeNode{
				Department:    d,
				Children:      build(d.ID),
				EmployeeCount: empCount[d.ID],
			})
		}
		return out
	}
	return build(rootParentID)
}

// DescendantIDs returns the given dept's ID + all descendant dept IDs.
// Used by chargeback rollups: "show me total spend under Engineering"
// includes every sub-team.
func (s *Store) DescendantIDs(deptID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []string{deptID}
	queue := []string{deptID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range s.Departments {
			if d.ParentID == cur {
				out = append(out, d.ID)
				queue = append(queue, d.ID)
			}
		}
	}
	return out
}

// --- Employee ops -------------------------------------------------------

// CreateEmployee validates and stores a new employee. Email is required;
// it's the natural human-readable identifier even before SSO is wired.
func (s *Store) CreateEmployee(e Employee) (*Employee, error) {
	if strings.TrimSpace(e.Email) == "" {
		return nil, fmt.Errorf("employee email required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.DepartmentID == "" {
		return nil, fmt.Errorf("departmentId required")
	}
	if _, ok := s.Departments[e.DepartmentID]; !ok {
		return nil, fmt.Errorf("department %q not found", e.DepartmentID)
	}
	// Reject duplicate email.
	for _, existing := range s.Employees {
		if strings.EqualFold(existing.Email, e.Email) {
			return nil, fmt.Errorf("email %q already registered as employee %q", e.Email, existing.ID)
		}
	}
	e.ID = s.nextID("emp")
	if e.Role == "" {
		e.Role = RoleEmployee
	}
	e.Active = true
	e.CreatedAt = time.Now()
	e.UpdatedAt = e.CreatedAt
	s.Employees[e.ID] = &e
	if err := s.saveLocked(); err != nil {
		delete(s.Employees, e.ID)
		return nil, err
	}
	return &e, nil
}

// UpdateEmployee patches mutable fields. SSOSubject can be set once
// (when the user first logs in via SSO) and is otherwise immutable.
func (s *Store) UpdateEmployee(patch Employee) (*Employee, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.Employees[patch.ID]
	if !ok {
		return nil, fmt.Errorf("employee %q not found", patch.ID)
	}
	if patch.DepartmentID != "" {
		if _, ok := s.Departments[patch.DepartmentID]; !ok {
			return nil, fmt.Errorf("department %q not found", patch.DepartmentID)
		}
		existing.DepartmentID = patch.DepartmentID
	}
	if strings.TrimSpace(patch.DisplayName) != "" {
		existing.DisplayName = patch.DisplayName
	}
	if patch.Role != "" {
		existing.Role = patch.Role
	}
	if patch.SSOSubject != "" && existing.SSOSubject == "" {
		existing.SSOSubject = patch.SSOSubject
	}
	if patch.ManagerID != "" {
		existing.ManagerID = patch.ManagerID
	}
	existing.UpdatedAt = time.Now()
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return existing, nil
}

// DeactivateEmployee flips Active=false. Preserves the record so audit
// trails and historical chargeback reports remain coherent.
func (s *Store) DeactivateEmployee(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.Employees[id]
	if !ok {
		return fmt.Errorf("employee %q not found", id)
	}
	e.Active = false
	e.UpdatedAt = time.Now()
	return s.saveLocked()
}

// FindEmployeeBySSO returns the employee bound to the given OIDC sub
// claim, or nil if none. Used by the SSO login flow to map an external
// identity to an internal record.
func (s *Store) FindEmployeeBySSO(sub string) *Employee {
	if sub == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.Employees {
		if e.SSOSubject == sub {
			copy := *e
			return &copy
		}
	}
	return nil
}

// FindEmployeeByEmail does case-insensitive match.
func (s *Store) FindEmployeeByEmail(email string) *Employee {
	if email == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.Employees {
		if strings.EqualFold(e.Email, email) {
			copy := *e
			return &copy
		}
	}
	return nil
}

// ListEmployees returns all employees, optionally filtered by department.
// activeOnly excludes deactivated employees.
func (s *Store) ListEmployees(deptID string, activeOnly bool) []Employee {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Employee, 0, len(s.Employees))
	for _, e := range s.Employees {
		if activeOnly && !e.Active {
			continue
		}
		if deptID != "" && e.DepartmentID != deptID {
			continue
		}
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DisplayName < out[j].DisplayName })
	return out
}

// --- helpers ------------------------------------------------------------

func (s *Store) nextID(prefix string) string {
	s.idCounter++
	return fmt.Sprintf("%s_%d", prefix, s.idCounter)
}
