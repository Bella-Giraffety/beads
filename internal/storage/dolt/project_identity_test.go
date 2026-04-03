package dolt

import (
	"strings"
	"testing"

	"github.com/steveyegge/beads/internal/configfile"
)

func TestNewFromConfig_ProjectIdentityMismatchMentionsStaleMetadata(t *testing.T) {
	skipIfNoDolt(t)
	acquireTestSlot()
	t.Cleanup(releaseTestSlot)

	ctx, cancel := testContext(t)
	defer cancel()

	beadsDir := t.TempDir()
	dbPath := t.TempDir()
	dbName := uniqueTestDBName(t)
	const liveProjectID = "proj-live"
	const staleProjectID = "proj-stale"

	store, err := New(ctx, &Config{
		Path:            dbPath,
		CommitterName:   "test",
		CommitterEmail:  "test@example.com",
		ServerHost:      "127.0.0.1",
		ServerPort:      testServerPort,
		Database:        dbName,
		CreateIfMissing: true,
	})
	if err != nil {
		t.Fatalf("create live database: %v", err)
	}
	defer store.Close()

	if err := store.SetMetadata(ctx, "_project_id", liveProjectID); err != nil {
		t.Fatalf("set live project id: %v", err)
	}

	fileCfg := &configfile.Config{
		Backend:        configfile.BackendDolt,
		DoltMode:       configfile.DoltModeServer,
		DoltServerHost: "127.0.0.1",
		DoltServerPort: testServerPort,
		DoltDatabase:   dbName,
		ProjectID:      staleProjectID,
	}
	if err := fileCfg.Save(beadsDir); err != nil {
		t.Fatalf("write metadata.json: %v", err)
	}

	staleStore, err := NewFromConfig(ctx, beadsDir)
	if staleStore != nil {
		staleStore.Close()
	}
	if err == nil {
		t.Fatal("expected project identity mismatch, got nil")
	}

	msg := err.Error()
	for _, want := range []string{
		"PROJECT IDENTITY MISMATCH",
		staleProjectID,
		liveProjectID,
		beadsDir,
		"Stale local metadata after DB regeneration/recovery (project-id drift)",
		"Confirm the live server/database: bd dolt status",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected mismatch error to contain %q, got:\n%s", want, msg)
		}
	}

	if strings.Contains(msg, "This means the Dolt server is serving a DIFFERENT project's database.") {
		t.Fatalf("mismatch error still reports only the old wrong-server diagnosis:\n%s", msg)
	}
}
