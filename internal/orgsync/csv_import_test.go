package orgsync

import (
	"strings"
	"testing"
)

func TestImportCSV_Headerful(t *testing.T) {
	s := newStore(t)
	eng, _ := s.CreateDepartment(Department{Name: "Engineering"})
	sales, _ := s.CreateDepartment(Department{Name: "Sales"})

	csv := strings.Join([]string{
		"email,display_name,department,role",
		"alice@x.com,Alice,Engineering,team_lead",
		"bob@x.com,Bob,Sales,",
		"charlie@x.com,,Sales,employee",
	}, "\n")

	r, err := s.ImportEmployeesCSV(csv, "")
	if err != nil {
		t.Fatal(err)
	}
	if r.Created != 3 {
		t.Errorf("expected 3 created, got %+v", r)
	}
	if len(r.ErrorRows) != 0 {
		t.Errorf("expected no errors, got %+v", r.ErrorRows)
	}
	a := s.FindEmployeeByEmail("alice@x.com")
	if a == nil || a.DepartmentID != eng.ID || a.Role != RoleTeamLead {
		t.Errorf("alice mis-imported: %+v", a)
	}
	b := s.FindEmployeeByEmail("bob@x.com")
	if b == nil || b.DepartmentID != sales.ID || b.Role != RoleEmployee {
		t.Errorf("bob mis-imported: %+v", b)
	}
}

func TestImportCSV_Headerless_UsesDefaultSchema(t *testing.T) {
	s := newStore(t)
	dept, _ := s.CreateDepartment(Department{Name: "X"})

	csv := "alice@x.com,Alice,X,employee\nbob@x.com,,X,"
	r, err := s.ImportEmployeesCSV(csv, dept.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Created != 2 {
		t.Errorf("expected 2 created, got %+v", r)
	}
}

func TestImportCSV_ResolvesByID(t *testing.T) {
	s := newStore(t)
	dept, _ := s.CreateDepartment(Department{Name: "Engineering"})
	csv := "email,department\nalice@x.com," + dept.ID
	r, _ := s.ImportEmployeesCSV(csv, "")
	if r.Created != 1 {
		t.Errorf("expected 1 created via dept id, got %+v", r)
	}
}

func TestImportCSV_DefaultDept(t *testing.T) {
	s := newStore(t)
	dept, _ := s.CreateDepartment(Department{Name: "Catchall"})
	// CSV without department column at all — relies on defaultDeptID.
	csv := "email,display_name\nalice@x.com,Alice\nbob@x.com,Bob"
	r, _ := s.ImportEmployeesCSV(csv, dept.ID)
	if r.Created != 2 {
		t.Errorf("expected 2 created, got %+v", r)
	}
}

func TestImportCSV_UpdatesExisting(t *testing.T) {
	s := newStore(t)
	a, _ := s.CreateDepartment(Department{Name: "A"})
	b, _ := s.CreateDepartment(Department{Name: "B"})
	s.CreateEmployee(Employee{Email: "alice@x.com", DepartmentID: a.ID, DisplayName: "Old"})

	csv := "email,display_name,department,role\nalice@x.com,Alice New,B,team_lead"
	r, _ := s.ImportEmployeesCSV(csv, "")
	if r.Updated != 1 || r.Created != 0 {
		t.Errorf("expected 1 updated/0 created, got %+v", r)
	}
	a2 := s.FindEmployeeByEmail("alice@x.com")
	if a2.DisplayName != "Alice New" || a2.DepartmentID != b.ID || a2.Role != RoleTeamLead {
		t.Errorf("update didn't apply: %+v", a2)
	}
}

func TestImportCSV_RejectsUnknownDept(t *testing.T) {
	s := newStore(t)
	csv := "email,department\nalice@x.com,Mars Branch"
	r, _ := s.ImportEmployeesCSV(csv, "")
	if r.Created != 0 || len(r.ErrorRows) != 1 {
		t.Errorf("expected 0/1, got %+v", r)
	}
	if r.ErrorRows[0].Email != "alice@x.com" {
		t.Errorf("error row email mismatch: %+v", r.ErrorRows[0])
	}
}

func TestImportCSV_RejectsBadRole(t *testing.T) {
	s := newStore(t)
	dept, _ := s.CreateDepartment(Department{Name: "X"})
	csv := "email,department,role\nalice@x.com,X,supreme_overlord"
	r, _ := s.ImportEmployeesCSV(csv, dept.ID)
	if r.Created != 0 || len(r.ErrorRows) != 1 {
		t.Errorf("expected reject bad role, got %+v", r)
	}
}

func TestImportCSV_ReactivatesInactive(t *testing.T) {
	s := newStore(t)
	dept, _ := s.CreateDepartment(Department{Name: "X"})
	emp, _ := s.CreateEmployee(Employee{Email: "alice@x.com", DepartmentID: dept.ID})
	s.DeactivateEmployee(emp.ID)

	csv := "email,department\nalice@x.com,X"
	r, _ := s.ImportEmployeesCSV(csv, "")
	if r.Updated != 1 {
		t.Errorf("expected re-activation as update, got %+v", r)
	}
	got := s.FindEmployeeByEmail("alice@x.com")
	if !got.Active {
		t.Error("expected employee to be re-activated")
	}
}
