package dolt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steveyegge/beads/internal/configfile"
)

func writeServerMetadata(t *testing.T, beadsDir, dbName, projectID string, sharedServer bool) {
	t.Helper()
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatalf("mkdir beads dir: %v", err)
	}
	cfg := &configfile.Config{
		Backend:        configfile.BackendDolt,
		Database:       "dolt",
		DoltDatabase:   dbName,
		DoltMode:       configfile.DoltModeServer,
		DoltServerHost: "127.0.0.1",
		DoltServerPort: testServerPort,
		ProjectID:      projectID,
	}
	if err := cfg.Save(beadsDir); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	sharedServerText := "false"
	if sharedServer {
		sharedServerText = "true"
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "config.yaml"), []byte(fmt.Sprintf("dolt:\n  shared-server: %s\n", sharedServerText)), 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
}

func createProjectIdentityDatabase(t *testing.T, ctx context.Context, beadsDir, dbName, projectID string) {
	t.Helper()
	writeServerMetadata(t, beadsDir, dbName, projectID, false)
	store, err := New(ctx, &Config{
		Path:            filepath.Join(beadsDir, "dolt"),
		BeadsDir:        beadsDir,
		Database:        dbName,
		ServerHost:      "127.0.0.1",
		ServerPort:      testServerPort,
		CreateIfMissing: true,
		MaxOpenConns:    1,
	})
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer store.Close()
	if err := store.SetMetadata(ctx, "_project_id", projectID); err != nil {
		t.Fatalf("set _project_id: %v", err)
	}
}

func TestNewFromConfig_RepairsSharedServerProjectIdentityDrift(t *testing.T) {
	t.Setenv("BEADS_DOLT_SHARED_SERVER", "")
	skipIfNoServer(t)

	ctx, cancel := testContext(t)
	defer cancel()

	dbName := uniqueTestDBName(t)
	authoritativeID := "db-project-id-shared"
	staleID := "stale-local-id"

	ownerBeadsDir := filepath.Join(t.TempDir(), ".beads")
	createProjectIdentityDatabase(t, ctx, ownerBeadsDir, dbName, authoritativeID)

	clientBeadsDir := filepath.Join(t.TempDir(), ".beads")
	writeServerMetadata(t, clientBeadsDir, dbName, staleID, true)

	store, err := NewFromConfig(ctx, clientBeadsDir)
	if err != nil {
		t.Fatalf("NewFromConfig() error = %v, want repair success", err)
	}
	defer store.Close()

	loaded, err := configfile.Load(clientBeadsDir)
	if err != nil {
		t.Fatalf("load repaired metadata: %v", err)
	}
	if loaded.ProjectID != authoritativeID {
		t.Fatalf("ProjectID = %q, want %q after repair", loaded.ProjectID, authoritativeID)
	}
	if err := store.verifyProjectIdentity(ctx, clientBeadsDir); err != nil {
		t.Fatalf("verifyProjectIdentity() after repair = %v, want nil", err)
	}
	if !shouldRepairSharedServerProjectIdentity(clientBeadsDir) {
		t.Fatal("shouldRepairSharedServerProjectIdentity() = false, want true")
	}
	if loaded.ProjectID == staleID {
		t.Fatal("stale project_id remained in metadata.json")
	}
}

func TestNewFromConfig_RejectsNonSharedServerProjectIdentityDrift(t *testing.T) {
	t.Setenv("BEADS_DOLT_SHARED_SERVER", "")
	skipIfNoServer(t)

	ctx, cancel := testContext(t)
	defer cancel()

	dbName := uniqueTestDBName(t)
	authoritativeID := "db-project-id-standalone"
	staleID := "stale-local-id"

	ownerBeadsDir := filepath.Join(t.TempDir(), ".beads")
	createProjectIdentityDatabase(t, ctx, ownerBeadsDir, dbName, authoritativeID)

	clientBeadsDir := filepath.Join(t.TempDir(), ".beads")
	writeServerMetadata(t, clientBeadsDir, dbName, staleID, false)

	_, err := NewFromConfig(ctx, clientBeadsDir)
	if err == nil {
		t.Fatal("NewFromConfig() error = nil, want PROJECT IDENTITY MISMATCH")
	}
	if !strings.Contains(err.Error(), "PROJECT IDENTITY MISMATCH") {
		t.Fatalf("error = %v, want PROJECT IDENTITY MISMATCH", err)
	}

	loaded, loadErr := configfile.Load(clientBeadsDir)
	if loadErr != nil {
		t.Fatalf("load metadata after failed open: %v", loadErr)
	}
	if loaded.ProjectID != staleID {
		t.Fatalf("ProjectID = %q, want %q without shared-server repair", loaded.ProjectID, staleID)
	}
	if shouldRepairSharedServerProjectIdentity(clientBeadsDir) {
		t.Fatal("shouldRepairSharedServerProjectIdentity() = true, want false")
	}
	if loaded.ProjectID == authoritativeID {
		t.Fatal("non-shared-server open unexpectedly rewrote metadata.json")
	}
}
