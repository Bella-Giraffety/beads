package dolt

import (
	"context"
	"testing"

	"github.com/steveyegge/beads/internal/storage/schema"
)

func TestNew_ReadOnlyRepairsMissingIgnoredTables(t *testing.T) {
	if testServerPort == 0 {
		t.Skip("Dolt test container not running")
	}

	ctx := context.Background()
	dbName := uniqueTestDBName(t)
	t.Cleanup(func() { dropTestDatabase(t, testServerPort, dbName) })

	initStore, err := New(ctx, &Config{
		Path:            t.TempDir(),
		ServerHost:      "127.0.0.1",
		ServerPort:      testServerPort,
		Database:        dbName,
		CreateIfMissing: true,
		MaxOpenConns:    1,
	})
	if err != nil {
		t.Fatalf("initial New() error = %v", err)
	}

	for _, table := range []string{"wisp_comments", "wisp_events", "wisp_dependencies", "wisp_labels", "wisps", "repo_mtimes", "local_metadata"} {
		if _, err := initStore.db.ExecContext(ctx, "DROP TABLE IF EXISTS "+table); err != nil {
			initStore.Close()
			t.Fatalf("drop %s: %v", table, err)
		}
	}
	if ok, err := schema.TableExists(ctx, initStore.db, "wisps"); err != nil {
		initStore.Close()
		t.Fatalf("precondition check wisps: %v", err)
	} else if ok {
		initStore.Close()
		t.Fatal("expected wisps table to be missing before read-only reopen")
	}
	if ok, err := schema.TableExists(ctx, initStore.db, "repo_mtimes"); err != nil {
		initStore.Close()
		t.Fatalf("precondition check repo_mtimes: %v", err)
	} else if ok {
		initStore.Close()
		t.Fatal("expected repo_mtimes table to be missing before read-only reopen")
	}
	if ok, err := schema.TableExists(ctx, initStore.db, "local_metadata"); err != nil {
		initStore.Close()
		t.Fatalf("precondition check local_metadata: %v", err)
	} else if ok {
		initStore.Close()
		t.Fatal("expected local_metadata table to be missing before read-only reopen")
	}
	initStore.Close()

	readOnlyStore, err := New(ctx, &Config{
		Path:         t.TempDir(),
		ServerHost:   "127.0.0.1",
		ServerPort:   testServerPort,
		Database:     dbName,
		ReadOnly:     true,
		MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("read-only New() error = %v", err)
	}
	defer readOnlyStore.Close()

	for _, table := range []string{"wisps", "repo_mtimes", "local_metadata"} {
		ok, err := schema.TableExists(ctx, readOnlyStore.db, table)
		if err != nil {
			t.Fatalf("TableExists(%s): %v", table, err)
		}
		if !ok {
			t.Fatalf("expected %s to be recreated on read-only open", table)
		}
	}
}
