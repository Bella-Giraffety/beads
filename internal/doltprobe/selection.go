package doltprobe

import "strings"

// Candidate describes a server database that may be considered during repair
// or diagnostics when the configured database is missing or unreadable.
type Candidate struct {
	Name           string
	HasIssuesTable bool
	ProjectID      string
}

// SelectAuthoritativeDatabase returns the only candidate that matches the
// expected project identity. Empty project IDs and ambiguous matches are
// rejected so stale compat/orphan catalogs are never treated as authoritative.
func SelectAuthoritativeDatabase(expectedProjectID string, candidates []Candidate) string {
	expectedProjectID = strings.TrimSpace(expectedProjectID)
	if expectedProjectID == "" {
		return ""
	}

	match := ""
	for _, candidate := range candidates {
		if !candidate.HasIssuesTable || strings.TrimSpace(candidate.ProjectID) != expectedProjectID {
			continue
		}
		if match != "" {
			return ""
		}
		match = candidate.Name
	}

	return match
}
