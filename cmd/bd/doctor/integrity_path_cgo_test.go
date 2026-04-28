//go:build cgo

package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/steveyegge/beads/internal/beads"
	"github.com/steveyegge/beads/internal/configfile"
	"github.com/steveyegge/beads/internal/storage/dolt"
)

func TestCheckRepoFingerprint_UsesTargetRepoOutsideCWD(t *testing.T) {
	outerRepo := t.TempDir()
	targetRepo := t.TempDir()

	setupGitRepoInDir(t, outerRepo)
	setupGitRepoInDir(t, targetRepo)

	targetRepoID, err := beads.ComputeRepoIDForPath(targetRepo)
	if err != nil {
		t.Fatalf("ComputeRepoIDForPath(targetRepo) failed: %v", err)
	}

	beadsDir := filepath.Join(targetRepo, ".beads")
	cfg := &configfile.Config{
		Database: "dolt",
		Backend:  configfile.BackendDolt,
	}
	if err := cfg.Save(beadsDir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	ctx := context.Background()
	store, err := dolt.New(ctx, &dolt.Config{
		Path:     filepath.Join(beadsDir, "dolt"),
		Database: "beads",
	})
	if err != nil {
		t.Skipf("skipping: Dolt server not available: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.SetMetadata(ctx, "repo_id", targetRepoID); err != nil {
		t.Fatalf("failed to set repo_id metadata: %v", err)
	}

	runInDir(t, outerRepo, func() {
		check := CheckRepoFingerprint(targetRepo)

		if check.Status != StatusOK {
			t.Fatalf("Status = %q, want %q (message=%q detail=%q)", check.Status, StatusOK, check.Message, check.Detail)
		}
		if check.Message != "Verified ("+targetRepoID[:8]+")" {
			t.Fatalf("Message = %q, want %q", check.Message, "Verified ("+targetRepoID[:8]+")")
		}
	})
}

func TestCheckRepoFingerprint_FollowsRedirectTargetRepo(t *testing.T) {
	outerRepo := t.TempDir()
	targetRepo := t.TempDir()

	setupGitRepoInDir(t, outerRepo)
	setupGitRepoInDir(t, targetRepo)
	runInDir(t, outerRepo, func() {
		runDoctorGitInDir(t, outerRepo, "remote", "add", "origin", "https://github.com/example/outer.git")
	})
	runInDir(t, targetRepo, func() {
		runDoctorGitInDir(t, targetRepo, "remote", "add", "origin", "https://github.com/example/target.git")
	})

	targetRepoID, err := beads.ComputeRepoIDForPath(targetRepo)
	if err != nil {
		t.Fatalf("ComputeRepoIDForPath(targetRepo) failed: %v", err)
	}

	targetBeadsDir := filepath.Join(targetRepo, ".beads")
	cfg := &configfile.Config{Database: "dolt", Backend: configfile.BackendDolt}
	if err := cfg.Save(targetBeadsDir); err != nil {
		t.Fatalf("failed to save target config: %v", err)
	}

	outerBeadsDir := filepath.Join(outerRepo, ".beads")
	if err := os.MkdirAll(outerBeadsDir, 0o755); err != nil {
		t.Fatalf("failed to create outer .beads: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outerBeadsDir, "redirect"), []byte(targetBeadsDir+"\n"), 0o644); err != nil {
		t.Fatalf("failed to write redirect: %v", err)
	}

	ctx := context.Background()
	store, err := dolt.New(ctx, &dolt.Config{Path: filepath.Join(targetBeadsDir, "dolt"), Database: "beads"})
	if err != nil {
		t.Skipf("skipping: Dolt server not available: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.SetMetadata(ctx, "repo_id", targetRepoID); err != nil {
		t.Fatalf("failed to set repo_id metadata: %v", err)
	}

	check := CheckRepoFingerprint(outerRepo)
	if check.Status != StatusOK {
		t.Fatalf("Status = %q, want %q (message=%q detail=%q)", check.Status, StatusOK, check.Message, check.Detail)
	}
	if check.Message != "Verified ("+targetRepoID[:8]+")" {
		t.Fatalf("Message = %q, want %q", check.Message, "Verified ("+targetRepoID[:8]+")")
	}
}

func TestCheckRepoFingerprint_DowngradesToWarningWhenProjectIdentityMatches(t *testing.T) {
	repoDir := t.TempDir()
	setupGitRepoInDir(t, repoDir)

	beadsDir := filepath.Join(repoDir, ".beads")
	cfg := &configfile.Config{
		Database:  "dolt",
		Backend:   configfile.BackendDolt,
		ProjectID: "project-123",
	}
	if err := cfg.Save(beadsDir); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	ctx := context.Background()
	store, err := dolt.New(ctx, &dolt.Config{Path: filepath.Join(beadsDir, "dolt"), Database: "beads"})
	if err != nil {
		t.Skipf("skipping: Dolt server not available: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.SetMetadata(ctx, "repo_id", "deadbeefdeadbeefdeadbeefdeadbeef"); err != nil {
		t.Fatalf("failed to set repo_id metadata: %v", err)
	}
	if err := store.SetMetadata(ctx, "_project_id", "project-123"); err != nil {
		t.Fatalf("failed to set project identity: %v", err)
	}

	check := CheckRepoFingerprint(repoDir)
	if check.Status != StatusWarning {
		t.Fatalf("Status = %q, want %q (message=%q detail=%q)", check.Status, StatusWarning, check.Message, check.Detail)
	}
	if check.Message != "Repository fingerprint drift detected" {
		t.Fatalf("Message = %q, want %q", check.Message, "Repository fingerprint drift detected")
	}
}
