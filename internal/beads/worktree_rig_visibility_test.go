package beads

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/steveyegge/beads/internal/git"
)

type rigWorktreeLayout struct {
	rigRoot         string
	canonicalBeads  string
	mayorRepo       string
	mayorBeads      string
	polecatRoot     string
	polecatLocalDir string
	polecatRepo     string
}

func setupCanonicalRigRedirectLayout(t *testing.T) rigWorktreeLayout {
	t.Helper()

	tmpDir := t.TempDir()
	rigRoot := filepath.Join(tmpDir, "rig")
	canonicalBeads := filepath.Join(rigRoot, ".beads")
	if err := os.MkdirAll(filepath.Join(canonicalBeads, "dolt"), 0o755); err != nil {
		t.Fatal(err)
	}

	mayorRepo := filepath.Join(rigRoot, "mayor", "beads")
	if err := os.MkdirAll(mayorRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := initGitRepoWithCommit(mayorRepo); err != nil {
		t.Fatalf("failed to init mayor repo: %v", err)
	}

	mayorBeads := filepath.Join(mayorRepo, ".beads")
	if err := os.MkdirAll(mayorBeads, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mayorBeads, "redirect"), []byte(canonicalBeads+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	branchCmd := exec.Command("git", "branch", "polecat-canonical-visibility")
	branchCmd.Dir = mayorRepo
	if out, err := branchCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create branch: %v\n%s", err, out)
	}

	polecatRoot := filepath.Join(rigRoot, "polecats", "quartz")
	polecatRepo := filepath.Join(polecatRoot, "beads")
	if err := os.MkdirAll(polecatRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	wtCmd := exec.Command("git", "worktree", "add", polecatRepo, "polecat-canonical-visibility")
	wtCmd.Dir = mayorRepo
	if out, err := wtCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create polecat worktree: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		cleanupCmd := exec.Command("git", "worktree", "remove", polecatRepo)
		cleanupCmd.Dir = mayorRepo
		_, _ = cleanupCmd.CombinedOutput()
	})

	polecatLocalDir := filepath.Join(polecatRoot, ".beads")
	if err := os.MkdirAll(polecatLocalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(polecatLocalDir, "redirect"), []byte(canonicalBeads+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	return rigWorktreeLayout{
		rigRoot:         rigRoot,
		canonicalBeads:  canonicalBeads,
		mayorRepo:       mayorRepo,
		mayorBeads:      mayorBeads,
		polecatRoot:     polecatRoot,
		polecatLocalDir: polecatLocalDir,
		polecatRepo:     polecatRepo,
	}
}

func setupRigWorktreeLayout(t *testing.T) rigWorktreeLayout {
	t.Helper()

	tmpDir := t.TempDir()
	rigRoot := filepath.Join(tmpDir, "rig")
	mayorRepo := filepath.Join(rigRoot, "mayor", "beads")
	if err := os.MkdirAll(mayorRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := initGitRepoWithCommit(mayorRepo); err != nil {
		t.Fatalf("failed to init mayor repo: %v", err)
	}

	mayorBeads := filepath.Join(mayorRepo, ".beads")
	if err := os.MkdirAll(filepath.Join(mayorBeads, "dolt"), 0o755); err != nil {
		t.Fatal(err)
	}

	canonicalBeads := filepath.Join(rigRoot, ".beads")
	if err := os.MkdirAll(filepath.Join(canonicalBeads, "dolt"), 0o755); err != nil {
		t.Fatal(err)
	}

	branchCmd := exec.Command("git", "branch", "polecat-rig-redirect")
	branchCmd.Dir = mayorRepo
	if out, err := branchCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create branch: %v\n%s", err, out)
	}

	polecatRoot := filepath.Join(rigRoot, "polecats", "quartz")
	polecatRepo := filepath.Join(polecatRoot, "beads")
	if err := os.MkdirAll(polecatRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	wtCmd := exec.Command("git", "worktree", "add", polecatRepo, "polecat-rig-redirect")
	wtCmd.Dir = mayorRepo
	if out, err := wtCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create polecat worktree: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		cleanupCmd := exec.Command("git", "worktree", "remove", polecatRepo)
		cleanupCmd.Dir = mayorRepo
		_, _ = cleanupCmd.CombinedOutput()
	})

	polecatLocalDir := filepath.Join(polecatRoot, ".beads")
	if err := os.MkdirAll(polecatLocalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(polecatLocalDir, "redirect"), []byte(canonicalBeads+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	return rigWorktreeLayout{
		rigRoot:         rigRoot,
		canonicalBeads:  canonicalBeads,
		mayorRepo:       mayorRepo,
		mayorBeads:      mayorBeads,
		polecatRoot:     polecatRoot,
		polecatLocalDir: polecatLocalDir,
		polecatRepo:     polecatRepo,
	}
}

func TestGetRepoContext_WorktreeAncestorRedirectBeatsMainRepoBeads(t *testing.T) {
	originalBeadsDir := os.Getenv("BEADS_DIR")
	t.Cleanup(func() {
		if originalBeadsDir != "" {
			os.Setenv("BEADS_DIR", originalBeadsDir)
		} else {
			os.Unsetenv("BEADS_DIR")
		}
		ResetCaches()
		git.ResetCaches()
	})
	os.Unsetenv("BEADS_DIR")

	layout := setupRigWorktreeLayout(t)
	t.Chdir(layout.polecatRepo)
	ResetCaches()
	git.ResetCaches()

	if got := findLocalBeadsDir(); got != layout.polecatLocalDir {
		t.Fatalf("findLocalBeadsDir() = %q, want local redirect dir %q", got, layout.polecatLocalDir)
	}

	info := GetRedirectInfo()
	if !info.IsRedirected {
		t.Fatal("GetRedirectInfo() should report redirected context from polecat worktree")
	}
	if info.LocalDir != layout.polecatLocalDir {
		t.Fatalf("GetRedirectInfo().LocalDir = %q, want %q", info.LocalDir, layout.polecatLocalDir)
	}
	if info.TargetDir != layout.canonicalBeads {
		t.Fatalf("GetRedirectInfo().TargetDir = %q, want %q", info.TargetDir, layout.canonicalBeads)
	}

	rc, err := GetRepoContext()
	if err != nil {
		t.Fatalf("GetRepoContext() failed: %v", err)
	}
	if got := FindBeadsDir(); got != layout.canonicalBeads {
		t.Fatalf("FindBeadsDir() = %q, want %q", got, layout.canonicalBeads)
	}
	if rc.BeadsDir != layout.canonicalBeads {
		t.Fatalf("RepoContext.BeadsDir = %q, want %q", rc.BeadsDir, layout.canonicalBeads)
	}
	if !rc.IsRedirected {
		t.Fatal("RepoContext.IsRedirected = false, want true")
	}
	if rc.CWDRepoRoot != layout.polecatRepo {
		t.Fatalf("RepoContext.CWDRepoRoot = %q, want %q", rc.CWDRepoRoot, layout.polecatRepo)
	}
}

func TestFindDatabasePath_WorktreeAncestorRedirectBeatsMainRepoDatabase(t *testing.T) {
	originalBeadsDir := os.Getenv("BEADS_DIR")
	originalBeadsDB := os.Getenv("BEADS_DB")
	t.Cleanup(func() {
		if originalBeadsDir != "" {
			os.Setenv("BEADS_DIR", originalBeadsDir)
		} else {
			os.Unsetenv("BEADS_DIR")
		}
		if originalBeadsDB != "" {
			os.Setenv("BEADS_DB", originalBeadsDB)
		} else {
			os.Unsetenv("BEADS_DB")
		}
		ResetCaches()
		git.ResetCaches()
	})
	os.Unsetenv("BEADS_DIR")
	os.Unsetenv("BEADS_DB")

	layout := setupRigWorktreeLayout(t)
	t.Chdir(layout.polecatRepo)
	ResetCaches()
	git.ResetCaches()

	got := FindDatabasePath()
	expected := filepath.Join(layout.canonicalBeads, "dolt")
	if got != expected {
		t.Fatalf("FindDatabasePath() = %q, want canonical rig database %q", got, expected)
	}
	wrong := filepath.Join(layout.mayorBeads, "dolt")
	if got == wrong {
		t.Fatalf("FindDatabasePath() picked mayor clone database %q instead of rig-local %q", wrong, expected)
	}
}

func TestCanonicalRigLocalBeadsVisibleFromMayorRigAndPolecatViews(t *testing.T) {
	originalBeadsDir := os.Getenv("BEADS_DIR")
	originalBeadsDB := os.Getenv("BEADS_DB")
	t.Cleanup(func() {
		if originalBeadsDir != "" {
			os.Setenv("BEADS_DIR", originalBeadsDir)
		} else {
			os.Unsetenv("BEADS_DIR")
		}
		if originalBeadsDB != "" {
			os.Setenv("BEADS_DB", originalBeadsDB)
		} else {
			os.Unsetenv("BEADS_DB")
		}
		ResetCaches()
		git.ResetCaches()
	})
	os.Unsetenv("BEADS_DIR")
	os.Unsetenv("BEADS_DB")

	layout := setupCanonicalRigRedirectLayout(t)
	expectedDB := filepath.Join(layout.canonicalBeads, "dolt")

	for _, dir := range []string{layout.mayorRepo, layout.rigRoot, layout.polecatRepo} {
		t.Chdir(dir)
		ResetCaches()
		git.ResetCaches()

		if got := FindBeadsDir(); got != layout.canonicalBeads {
			t.Fatalf("FindBeadsDir() from %s = %q, want %q", dir, got, layout.canonicalBeads)
		}
		if got := FindDatabasePath(); got != expectedDB {
			t.Fatalf("FindDatabasePath() from %s = %q, want %q", dir, got, expectedDB)
		}
	}
}
