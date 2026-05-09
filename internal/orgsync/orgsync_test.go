package orgsync

import (
	"strings"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCreateDepartment_RootAndChild(t *testing.T) {
	s := newStore(t)

	root, err := s.CreateDepartment(Department{Name: "Engineering", CostCenter: "ENG-001"})
	if err != nil {
		t.Fatal(err)
	}
	if root.ID == "" {
		t.Error("expected ID assigned")
	}

	child, err := s.CreateDepartment(Department{Name: "Platform", ParentID: root.ID})
	if err != nil {
		t.Fatal(err)
	}
	if child.ParentID != root.ID {
		t.Errorf("parent = %s, want %s", child.ParentID, root.ID)
	}
	if child.CostCenter != child.ID {
		t.Errorf("default cost-center should mirror id, got %s", child.CostCenter)
	}
}

func TestCreateDepartment_RejectsMissingParent(t *testing.T) {
	s := newStore(t)
	_, err := s.CreateDepartment(Department{Name: "Orphan", ParentID: "nonexistent"})
	if err == nil {
		t.Error("expected error for missing parent")
	}
}

func TestCreateDepartment_RejectsEmptyName(t *testing.T) {
	s := newStore(t)
	_, err := s.CreateDepartment(Department{Name: ""})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestUpdateDepartment_DetectsCycle(t *testing.T) {
	s := newStore(t)
	a, _ := s.CreateDepartment(Department{Name: "A"})
	b, _ := s.CreateDepartment(Department{Name: "B", ParentID: a.ID})
	c, _ := s.CreateDepartment(Department{Name: "C", ParentID: b.ID})

	// Try to make A a child of C — that would cycle A→C→B→A.
	_, err := s.UpdateDepartment(Department{ID: a.ID, ParentID: c.ID})
	if err == nil {
		t.Error("expected cycle detection")
	}
}

func TestDeleteDepartment_RejectsWithChildren(t *testing.T) {
	s := newStore(t)
	parent, _ := s.CreateDepartment(Department{Name: "Parent"})
	_, _ = s.CreateDepartment(Department{Name: "Child", ParentID: parent.ID})

	if err := s.DeleteDepartment(parent.ID); err == nil {
		t.Error("expected error — has children")
	}
}

func TestCreateEmployee_HappyPath(t *testing.T) {
	s := newStore(t)
	dept, _ := s.CreateDepartment(Department{Name: "Engineering"})
	e, err := s.CreateEmployee(Employee{
		Email:        "marvin@example.com",
		DisplayName:  "Marvin",
		DepartmentID: dept.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if e.Role != RoleEmployee {
		t.Errorf("default role = %s, want employee", e.Role)
	}
	if !e.Active {
		t.Error("new employee should be Active=true")
	}
}

func TestCreateEmployee_RejectsDuplicateEmail(t *testing.T) {
	s := newStore(t)
	dept, _ := s.CreateDepartment(Department{Name: "X"})
	s.CreateEmployee(Employee{Email: "a@b.com", DepartmentID: dept.ID})
	_, err := s.CreateEmployee(Employee{Email: "A@b.com", DepartmentID: dept.ID})
	if err == nil {
		t.Error("expected duplicate-email error (case-insensitive)")
	}
}

func TestFindEmployeeBySSO(t *testing.T) {
	s := newStore(t)
	dept, _ := s.CreateDepartment(Department{Name: "X"})
	e, _ := s.CreateEmployee(Employee{Email: "u@x.com", DepartmentID: dept.ID})
	s.UpdateEmployee(Employee{ID: e.ID, SSOSubject: "okta-sub-12345"})

	got := s.FindEmployeeBySSO("okta-sub-12345")
	if got == nil || got.Email != "u@x.com" {
		t.Errorf("FindEmployeeBySSO got %+v", got)
	}
}

func TestUpdateEmployee_SSOSubjectImmutableOnceSet(t *testing.T) {
	s := newStore(t)
	dept, _ := s.CreateDepartment(Department{Name: "X"})
	e, _ := s.CreateEmployee(Employee{Email: "u@x.com", DepartmentID: dept.ID})

	s.UpdateEmployee(Employee{ID: e.ID, SSOSubject: "first-sub"})
	s.UpdateEmployee(Employee{ID: e.ID, SSOSubject: "tampered-sub"})

	got, _ := s.UpdateEmployee(Employee{ID: e.ID, DisplayName: "x"})
	if got.SSOSubject != "first-sub" {
		t.Errorf("SSOSubject should be locked to first-sub, got %s", got.SSOSubject)
	}
}

func TestDeactivateEmployee_PreservesRecord(t *testing.T) {
	s := newStore(t)
	dept, _ := s.CreateDepartment(Department{Name: "X"})
	e, _ := s.CreateEmployee(Employee{Email: "u@x.com", DepartmentID: dept.ID})
	if err := s.DeactivateEmployee(e.ID); err != nil {
		t.Fatal(err)
	}
	got := s.FindEmployeeByEmail("u@x.com")
	if got == nil {
		t.Fatal("record should remain")
	}
	if got.Active {
		t.Error("expected Active=false")
	}
}

func TestTree_BuildsHierarchy(t *testing.T) {
	s := newStore(t)
	root1, _ := s.CreateDepartment(Department{Name: "Engineering"})
	_, _ = s.CreateDepartment(Department{Name: "Platform", ParentID: root1.ID})
	_, _ = s.CreateDepartment(Department{Name: "Mobile", ParentID: root1.ID})
	root2, _ := s.CreateDepartment(Department{Name: "Sales"})

	tree := s.Tree()
	if len(tree) != 2 {
		t.Errorf("expected 2 roots, got %d", len(tree))
	}
	for _, n := range tree {
		if n.Department.ID == root1.ID && len(n.Children) != 2 {
			t.Errorf("Engineering should have 2 children, got %d", len(n.Children))
		}
		if n.Department.ID == root2.ID && len(n.Children) != 0 {
			t.Errorf("Sales should have 0 children, got %d", len(n.Children))
		}
	}
}

func TestDescendantIDs(t *testing.T) {
	s := newStore(t)
	root, _ := s.CreateDepartment(Department{Name: "Engineering"})
	platform, _ := s.CreateDepartment(Department{Name: "Platform", ParentID: root.ID})
	_, _ = s.CreateDepartment(Department{Name: "Storage", ParentID: platform.ID})
	_, _ = s.CreateDepartment(Department{Name: "Mobile", ParentID: root.ID})

	ids := s.DescendantIDs(root.ID)
	if len(ids) != 4 {
		t.Errorf("expected 4 descendants (incl self), got %d: %v", len(ids), ids)
	}
}

func TestPersists_AcrossStoreInstance(t *testing.T) {
	dir := t.TempDir()
	s1, _ := NewStore(dir)
	dept, _ := s1.CreateDepartment(Department{Name: "Engineering"})
	s1.CreateEmployee(Employee{Email: "a@x.com", DepartmentID: dept.ID})

	s2, _ := NewStore(dir)
	if got := s2.FindEmployeeByEmail("a@x.com"); got == nil {
		t.Error("expected hydrated employee after store restart")
	}
	depts := s2.ListDepartments()
	if len(depts) != 1 || depts[0].Name != "Engineering" {
		t.Errorf("expected 1 dept Engineering, got %+v", depts)
	}
}

func TestListEmployees_FilterByDept(t *testing.T) {
	s := newStore(t)
	a, _ := s.CreateDepartment(Department{Name: "A"})
	b, _ := s.CreateDepartment(Department{Name: "B"})
	s.CreateEmployee(Employee{Email: "1@x.com", DepartmentID: a.ID})
	s.CreateEmployee(Employee{Email: "2@x.com", DepartmentID: a.ID})
	s.CreateEmployee(Employee{Email: "3@x.com", DepartmentID: b.ID})

	got := s.ListEmployees(a.ID, false)
	if len(got) != 2 {
		t.Errorf("expected 2 in dept A, got %d", len(got))
	}
	for _, e := range got {
		if !strings.HasSuffix(e.Email, "@x.com") {
			t.Errorf("unexpected email %s", e.Email)
		}
	}
}
