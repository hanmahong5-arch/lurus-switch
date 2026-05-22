package orgsync

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
)

// CSVImportResult summarises a bulk import. Per-row errors don't fail
// the whole call — they're collected so the admin can fix the bad
// rows and re-paste; everything that parsed cleanly is committed.
type CSVImportResult struct {
	Created   int              `json:"created"`
	Updated   int              `json:"updated"`
	Skipped   int              `json:"skipped"`
	ErrorRows []CSVImportError `json:"errorRows"`
}

// CSVImportError describes one failed row.
type CSVImportError struct {
	LineNumber int    `json:"lineNumber"`
	Email      string `json:"email"`
	Reason     string `json:"reason"`
}

// ImportEmployeesCSV bulk-loads employees from a CSV string. Acts as a
// SCIM-lite for resellers who don't yet have an SCIM-compliant IDP —
// they paste a Workday/spreadsheet export and we provision the org
// chart.
//
// Expected columns (header row optional, case-insensitive):
//
//	email          (required)
//	display_name   (optional; "name" is also accepted)
//	department     (department name OR id; required unless defaultDeptID is set)
//	role           (optional; defaults to "employee" — must be one of the orgsync.Role constants)
//
// Rules:
//   - Existing employee with same email → DisplayName / DepartmentID /
//     Role are patched and the row is counted as Updated. SSO subject is
//     never altered by import (immutable once bound).
//   - Email casing is normalised (case-insensitive match).
//   - Unknown department → row recorded as ErrorRow, others continue.
func (s *Store) ImportEmployeesCSV(content string, defaultDeptID string) (CSVImportResult, error) {
	r := csv.NewReader(strings.NewReader(content))
	r.FieldsPerRecord = -1 // tolerant: rows may have trailing extras
	r.TrimLeadingSpace = true

	var result CSVImportResult
	headers := []string{}
	lineNum := 0

	for {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return result, fmt.Errorf("csv parse: %w", err)
		}
		lineNum++

		// Detect header on first row by looking for "email" anywhere.
		if lineNum == 1 && rowLooksLikeHeader(row) {
			headers = make([]string, len(row))
			for i, h := range row {
				headers[i] = strings.ToLower(strings.TrimSpace(h))
			}
			continue
		}
		if len(headers) == 0 {
			// Default schema if no header was provided.
			headers = []string{"email", "display_name", "department", "role"}
		}

		fields := mapRow(headers, row)
		email := strings.TrimSpace(fields["email"])
		if email == "" {
			result.ErrorRows = append(result.ErrorRows, CSVImportError{
				LineNumber: lineNum, Email: "", Reason: "missing email",
			})
			continue
		}
		display := strings.TrimSpace(firstNonEmpty(fields["display_name"], fields["name"]))
		deptRef := strings.TrimSpace(fields["department"])
		role := strings.TrimSpace(strings.ToLower(fields["role"]))

		deptID, err := s.resolveDept(deptRef, defaultDeptID)
		if err != nil {
			result.ErrorRows = append(result.ErrorRows, CSVImportError{
				LineNumber: lineNum, Email: email, Reason: err.Error(),
			})
			continue
		}

		var roleValue Role = RoleEmployee
		if role != "" {
			if !isKnownRole(role) {
				result.ErrorRows = append(result.ErrorRows, CSVImportError{
					LineNumber: lineNum, Email: email,
					Reason: fmt.Sprintf("unknown role %q", role),
				})
				continue
			}
			roleValue = Role(role)
		}

		// Try update existing by email first.
		existing := s.FindEmployeeByEmail(email)
		if existing != nil {
			if !existing.Active {
				// Re-activating an existing record is the right behavior
				// — the alternative is silently leaving the inactive flag
				// on, which makes import feel broken.
				if err := s.ReactivateEmployee(existing.ID); err != nil {
					result.ErrorRows = append(result.ErrorRows, CSVImportError{
						LineNumber: lineNum, Email: email, Reason: err.Error(),
					})
					continue
				}
			}
			patch := Employee{
				ID:           existing.ID,
				DisplayName:  display,
				DepartmentID: deptID,
				Role:         roleValue,
			}
			if _, err := s.UpdateEmployee(patch); err != nil {
				result.ErrorRows = append(result.ErrorRows, CSVImportError{
					LineNumber: lineNum, Email: email, Reason: err.Error(),
				})
				continue
			}
			result.Updated++
			continue
		}

		_, err = s.CreateEmployee(Employee{
			Email:        email,
			DisplayName:  display,
			DepartmentID: deptID,
			Role:         roleValue,
		})
		if err != nil {
			result.ErrorRows = append(result.ErrorRows, CSVImportError{
				LineNumber: lineNum, Email: email, Reason: err.Error(),
			})
			continue
		}
		result.Created++
	}

	if result.Created+result.Updated == 0 && len(result.ErrorRows) == 0 {
		result.Skipped++ // empty input — surface a non-zero count so UI shows "nothing happened"
	}
	return result, nil
}

// resolveDept matches a CSV department reference (name or id) to a
// known department. Falls back to defaultDeptID when ref is empty.
func (s *Store) resolveDept(ref, defaultDeptID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ref == "" {
		if defaultDeptID == "" {
			return "", fmt.Errorf("department required (and no default set)")
		}
		if _, ok := s.Departments[defaultDeptID]; !ok {
			return "", fmt.Errorf("default department %q not found", defaultDeptID)
		}
		return defaultDeptID, nil
	}
	// Try ID match first (cheap).
	if _, ok := s.Departments[ref]; ok {
		return ref, nil
	}
	// Then case-insensitive name match.
	for id, d := range s.Departments {
		if strings.EqualFold(d.Name, ref) {
			return id, nil
		}
	}
	return "", fmt.Errorf("department %q not found", ref)
}

// rowLooksLikeHeader checks if any cell mentions "email" — cheap
// heuristic to detect a header row vs a data row whose first column
// happens to look like an email.
func rowLooksLikeHeader(row []string) bool {
	for _, cell := range row {
		if strings.EqualFold(strings.TrimSpace(cell), "email") {
			return true
		}
	}
	return false
}

// mapRow zips headers with values into a name-keyed map. Missing
// trailing columns just yield empty strings — common when callers
// omit optional fields.
func mapRow(headers, row []string) map[string]string {
	m := make(map[string]string, len(headers))
	for i, h := range headers {
		if i < len(row) {
			m[h] = row[i]
		} else {
			m[h] = ""
		}
	}
	return m
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func isKnownRole(r string) bool {
	switch Role(r) {
	case RoleEmployee, RoleTeamLead, RoleDeptAdmin, RoleITAdmin, RoleCompliance, RoleFinance:
		return true
	}
	return false
}
