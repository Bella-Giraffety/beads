package schema

import (
	"context"
	"fmt"
)

var ignoredTables = []string{
	"wisp_dependencies",
	"wisp_events",
	"wisp_comments",
	"wisp_labels",
	"wisps",
}

// EnsureIgnoredTables checks whether the dolt_ignore'd tables exist in the
// current working set and creates them if any are missing. This is the fast
// path called after branch creation, checkout, and on session init.
//
// dolt_ignore entries are committed and persist across branches; only the
// tables themselves (which live in the working set) need recreation.
func EnsureIgnoredTables(ctx context.Context, db DBConn) error {
	for _, table := range requiredIgnoredTables {
		tableOK, err := TableExists(ctx, db, table)
		if err != nil {
			return fmt.Errorf("check %s table: %w", table, err)
		}
		if !tableOK {
			return CreateIgnoredTables(ctx, db)
		}
	}
	return nil
}

// CreateIgnoredTables unconditionally creates all dolt_ignore'd tables.
// All statements use CREATE/ALTER forms that are safe to re-run.
//
// This does NOT set up dolt_ignore entries or commit; those are migration
// concerns handled separately during bd init.
func CreateIgnoredTables(ctx context.Context, db DBConn) error {
	for _, ddl := range IgnoredTableDDL() {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			// Tolerate concurrent bootstrap races when some ignored tables already
			// exist and later ALTER/INDEX statements become no-ops from our view.
			if !isConcurrentInitError(err) {
				return fmt.Errorf("create ignored table: %w", err)
			}
		}
	}
	return nil
}

// RepairIgnoredTables drops and recreates the dolt_ignore'd wisp tables.
//
// This is the safe repair path for ignored-table working-set corruption: these
// tables do not participate in Dolt history, so rebuilding them repairs the
// local/session state without touching committed issue history.
func RepairIgnoredTables(ctx context.Context, db DBConn) error {
	for _, table := range ignoredTables {
		if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS "+table); err != nil { //nolint:gosec // G201: table names come from internal constants
			return fmt.Errorf("drop ignored table %s: %w", table, err)
		}
	}
	if err := CreateIgnoredTables(ctx, db); err != nil {
		return fmt.Errorf("recreate ignored tables: %w", err)
	}
	return nil
}

// TableExists checks if a table exists using SHOW TABLES LIKE.
// Uses SHOW TABLES rather than information_schema to avoid crashes when the
// Dolt server catalog contains stale database entries from cleaned-up
// worktrees (GH#2051). SHOW TABLES is inherently scoped to the current
// database.
func TableExists(ctx context.Context, db DBConn, table string) (bool, error) {
	// Use string interpolation because Dolt doesn't support prepared-statement
	// parameters for SHOW commands. Table names come from internal constants.
	// #nosec G202 -- table names come from internal constants, not user input.
	rows, err := db.QueryContext(ctx, "SHOW TABLES LIKE '"+table+"'") //nolint:gosec // G202: table name is an internal constant
	if err != nil {
		return false, fmt.Errorf("check table %s: %w", table, err)
	}
	defer rows.Close()
	return rows.Next(), nil
}
