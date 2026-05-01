package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/steveyegge/beads/internal/configfile"
)

type fakeMetadataStore struct {
	projectID string
	err       error
}

func (f fakeMetadataStore) GetMetadata(_ context.Context, key string) (string, error) {
	if key != "_project_id" {
		return "", nil
	}
	return f.projectID, f.err
}

func TestCurrentWorkspaceIdentity(t *testing.T) {
	beadsDir := filepath.Join(t.TempDir(), ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := (&configfile.Config{ProjectID: "project-a"}).Save(beadsDir); err != nil {
		t.Fatalf("save metadata.json: %v", err)
	}

	t.Run("mismatch detected", func(t *testing.T) {
		identity := currentWorkspaceIdentity(context.Background(), beadsDir, fakeMetadataStore{projectID: "project-b"})
		if !identity.Mismatch {
			t.Fatal("expected mismatch")
		}
		if identity.LocalID != "project-a" || identity.DatabaseID != "project-b" {
			t.Fatalf("unexpected identity status: %+v", identity)
		}
	})

	t.Run("missing database id does not mismatch", func(t *testing.T) {
		identity := currentWorkspaceIdentity(context.Background(), beadsDir, fakeMetadataStore{})
		if identity.Mismatch {
			t.Fatalf("expected no mismatch, got %+v", identity)
		}
		if identity.LocalID != "project-a" || identity.DatabaseID != "" {
			t.Fatalf("unexpected identity status: %+v", identity)
		}
	})
}

func TestValidateWorkspaceIdentity_NilStore(t *testing.T) {
	// When store is nil, validateWorkspaceIdentity should be a no-op
	// (no panic, no os.Exit)
	origStore := store
	store = nil
	defer func() { store = nil; store = origStore }()

	validateWorkspaceIdentity(nil, "/nonexistent")
	// If we got here, no os.Exit was called — pass
}

func TestValidateWorkspaceIdentity_NonexistentDir(t *testing.T) {
	// When beadsDir doesn't exist, configfile.Load fails and we skip validation
	origStore := store
	store = nil
	defer func() { store = origStore }()

	validateWorkspaceIdentity(nil, "/nonexistent/path/that/does/not/exist")
}
