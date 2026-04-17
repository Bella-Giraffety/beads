package schema

import (
	"fmt"
	"io/fs"
	"strings"
	"sync"
)

// ignoredMigration identifies an embedded .up.sql migration (or a subset of
// its statements) that defines or alters a dolt-ignored table.
type ignoredMigration struct {
	version int
	// filter, if non-empty, selects only statements containing this substring
	// (case-insensitive). When empty, the entire migration file is used.
	filter string
}

// ignoredMigrations lists the migrations that define or alter dolt-ignored
// tables, in the order they must be applied. The embedded .up.sql files are
// the single source of truth for the recreated ignored-table schema.
var ignoredMigrations = []ignoredMigration{
	{version: 11},                  // CREATE TABLE repo_mtimes
	{version: 20},                  // CREATE TABLE wisps
	{version: 21},                  // CREATE TABLE wisp_labels, wisp_dependencies, wisp_events, wisp_comments
	{version: 22},                  // CREATE INDEX on wisp_dependencies
	{version: 23, filter: "wisps"}, // ALTER TABLE wisps ADD COLUMN no_history (skip issues ALTER)
	{version: 27, filter: "wisps"}, // ALTER TABLE wisps ADD COLUMN started_at (skip issues ALTER)
}

var requiredIgnoredTables = []string{
	"repo_mtimes",
	"wisps",
	"wisp_labels",
	"wisp_dependencies",
	"wisp_events",
	"wisp_comments",
}

// These exported statements preserve existing migration call sites while
// keeping the embedded SQL migrations as the single schema source of truth.
var (
	WispsTableSchema       = mustFindMigrationStatement(20, "wisps")
	WispLabelsSchema       = mustFindMigrationStatement(21, "wisp_labels")
	WispDependenciesSchema = mustFindMigrationStatement(21, "wisp_dependencies")
	WispEventsSchema       = mustFindMigrationStatement(21, "wisp_events")
	WispCommentsSchema     = mustFindMigrationStatement(21, "wisp_comments")
)

var (
	ignoredDDLOnce sync.Once
	ignoredDDLVal  []string
)

// IgnoredTableDDL returns the ordered SQL needed to recreate every ignored
// table from embedded migrations. The result is cached after first build.
func IgnoredTableDDL() []string {
	ignoredDDLOnce.Do(func() {
		ignoredDDLVal = buildIgnoredTableDDL()
	})
	return ignoredDDLVal
}

func buildIgnoredTableDDL() []string {
	var result []string
	for _, im := range ignoredMigrations {
		raw := ReadMigrationSQL(im.version)
		stmts := splitStatements(raw)
		if im.filter != "" {
			filterLower := strings.ToLower(im.filter)
			for _, stmt := range stmts {
				if strings.Contains(strings.ToLower(stmt), filterLower) {
					result = append(result, stmt)
				}
			}
			continue
		}
		result = append(result, stmts...)
	}
	return result
}

func mustFindMigrationStatement(version int, contains string) string {
	contains = strings.ToLower(contains)
	for _, stmt := range splitStatements(ReadMigrationSQL(version)) {
		if strings.Contains(strings.ToLower(stmt), contains) {
			return stmt
		}
	}
	panic(fmt.Sprintf("schema: migration %04d missing statement containing %q", version, contains))
}

// ReadMigrationSQL reads the embedded .up.sql file for the given version.
// It panics if the migration is missing because this is a programmer error.
func ReadMigrationSQL(version int) string {
	entries, err := fs.ReadDir(upMigrations, "migrations")
	if err != nil {
		panic(fmt.Sprintf("schema: reading migrations dir: %v", err))
	}
	prefix := fmt.Sprintf("%04d_", version)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), ".up.sql") {
			data, err := upMigrations.ReadFile("migrations/" + entry.Name())
			if err != nil {
				panic(fmt.Sprintf("schema: reading migration %s: %v", entry.Name(), err))
			}
			return string(data)
		}
	}
	panic(fmt.Sprintf("schema: migration %04d not found", version))
}

// splitStatements splits SQL text on semicolons into non-empty statements.
func splitStatements(sql string) []string {
	raw := strings.Split(sql, ";")
	var out []string
	for _, stmt := range raw {
		stmt = stripSQLComments(stmt)
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			out = append(out, stmt)
		}
	}
	return out
}

// stripSQLComments removes lines starting with -- from SQL text.
func stripSQLComments(sql string) string {
	var lines []string
	for _, line := range strings.Split(sql, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
