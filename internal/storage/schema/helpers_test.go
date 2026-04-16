package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
)

type stubResult struct{}

func (stubResult) LastInsertId() (int64, error) { return 0, nil }
func (stubResult) RowsAffected() (int64, error) { return 0, nil }

type recordingDB struct {
	queries []string
	errFor  map[string]error
}

func (db *recordingDB) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	db.queries = append(db.queries, strings.TrimSpace(query))
	if err := db.errFor[query]; err != nil {
		return nil, err
	}
	return stubResult{}, nil
}

func (db *recordingDB) QueryContext(_ context.Context, _ string, _ ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("unexpected query")
}

func (db *recordingDB) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row {
	panic("unexpected query row")
}

func TestRepairIgnoredTables(t *testing.T) {
	db := &recordingDB{}
	if err := RepairIgnoredTables(context.Background(), db); err != nil {
		t.Fatalf("RepairIgnoredTables() error = %v", err)
	}

	if len(db.queries) < len(ignoredTables) {
		t.Fatalf("expected at least %d statements, got %d", len(ignoredTables), len(db.queries))
	}

	for i, table := range ignoredTables {
		want := "DROP TABLE IF EXISTS " + table
		if db.queries[i] != want {
			t.Fatalf("statement %d = %q, want %q", i, db.queries[i], want)
		}
	}

	if got := db.queries[len(ignoredTables)]; !strings.Contains(got, "CREATE TABLE IF NOT EXISTS local_metadata") {
		t.Fatalf("expected recreate statements after drops, got %q", got)
	}
}

func TestRepairIgnoredTables_DropFailure(t *testing.T) {
	db := &recordingDB{
		errFor: map[string]error{
			"DROP TABLE IF EXISTS wisps": fmt.Errorf("boom"),
		},
	}

	err := RepairIgnoredTables(context.Background(), db)
	if err == nil {
		t.Fatal("RepairIgnoredTables() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "drop ignored table wisps") {
		t.Fatalf("expected drop failure context, got %v", err)
	}
}
