package schema

import (
	"strings"
	"testing"
)

func TestIgnoredTableDDL(t *testing.T) {
	ddl := IgnoredTableDDL()
	if len(ddl) == 0 {
		t.Fatal("IgnoredTableDDL returned no statements")
	}

	combined := strings.Join(ddl, "\n")

	for _, table := range []string{
		"local_metadata",
		"repo_mtimes",
		"wisps",
		"wisp_labels",
		"wisp_dependencies",
		"wisp_events",
		"wisp_comments",
	} {
		if !strings.Contains(combined, table) {
			t.Errorf("IgnoredTableDDL missing reference to table %q", table)
		}
	}

	for _, col := range []string{"no_history"} {
		if !strings.Contains(combined, col) {
			t.Errorf("IgnoredTableDDL missing column %q", col)
		}
	}

	if !strings.Contains(combined, "idx_repo_mtimes_checked") {
		t.Error("IgnoredTableDDL missing idx_repo_mtimes_checked index")
	}
	if !strings.Contains(combined, "idx_wisp_events_created_at") {
		t.Error("IgnoredTableDDL missing idx_wisp_events_created_at index")
	}
}

func TestRequiredIgnoredTables(t *testing.T) {
	for _, table := range []string{
		"local_metadata",
		"repo_mtimes",
		"wisps",
		"wisp_labels",
		"wisp_dependencies",
		"wisp_events",
		"wisp_comments",
	} {
		found := false
		for _, candidate := range requiredIgnoredTables {
			if candidate == table {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("requiredIgnoredTables missing %q", table)
		}
	}
}
