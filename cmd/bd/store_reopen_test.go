//go:build cgo

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/configfile"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/types"
	"github.com/steveyegge/beads/internal/utils"
)

func TestWithStorage_ReopensUsingMetadata(t *testing.T) {
	ctx := context.Background()
	testDBPath := filepath.Join(t.TempDir(), "dolt")
	newTestStoreIsolatedDB(t, testDBPath, "cfg")

	var gotPrefix string
	err := withStorage(ctx, nil, testDBPath, func(s storage.DoltStorage) error {
		var err error
		gotPrefix, err = s.GetConfig(ctx, "issue_prefix")
		return err
	})
	if err != nil {
		t.Fatalf("withStorage() error = %v", err)
	}
	if gotPrefix != "cfg" {
		t.Fatalf("issue_prefix = %q, want %q", gotPrefix, "cfg")
	}
}

func TestResolveBeadsDirForDBPath_UsesRawBeadsDirForSymlinkedDBPath(t *testing.T) {
	repoDir := t.TempDir()
	beadsDir := filepath.Join(repoDir, ".beads")
	actualDBPath := filepath.Join(repoDir, "external-dolt")
	linkDBPath := filepath.Join(beadsDir, "dolt")

	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatalf("mkdir beads dir: %v", err)
	}
	if err := os.MkdirAll(actualDBPath, 0o755); err != nil {
		t.Fatalf("mkdir external dolt dir: %v", err)
	}
	if err := os.Symlink(actualDBPath, linkDBPath); err != nil {
		t.Fatalf("symlink db path: %v", err)
	}

	cfg := &configfile.Config{
		Database: "dolt",
		Backend:  configfile.BackendDolt,
	}
	if err := cfg.Save(beadsDir); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	if got := resolveBeadsDirForDBPath(linkDBPath); !utils.PathsEqual(got, beadsDir) {
		t.Fatalf("resolveBeadsDirForDBPath(%q) = %q, want %q", linkDBPath, got, beadsDir)
	}
}

func TestWithStorage_ReopensRedirectedSourceDatabaseAndPreservesEphemeralState(t *testing.T) {
	if testDoltServerPort == 0 {
		t.Skip("Dolt test server not available")
	}

	t.Setenv("BEADS_DOLT_SERVER_DATABASE", "")

	ctx := context.Background()
	repoDir := t.TempDir()
	sourceBeadsDir := filepath.Join(repoDir, "rig", ".beads")
	targetBeadsDir := filepath.Join(repoDir, "shared", ".beads")
	sourceDBPath := filepath.Join(sourceBeadsDir, "dolt")
	targetDBPath := filepath.Join(targetBeadsDir, "dolt")

	sourceStore := newTestStoreIsolatedDB(t, sourceDBPath, "rig")
	_ = newTestStoreIsolatedDB(t, targetDBPath, "shared")

	sourceCfg, err := configfile.Load(sourceBeadsDir)
	if err != nil {
		t.Fatalf("load source metadata: %v", err)
	}
	targetCfg, err := configfile.Load(targetBeadsDir)
	if err != nil {
		t.Fatalf("load target metadata: %v", err)
	}
	if sourceCfg.GetDoltDatabase() == targetCfg.GetDoltDatabase() {
		t.Fatal("expected source and target databases to differ")
	}

	root := &types.Issue{
		ID:        "startup-root",
		Title:     "Startup thread root",
		Status:    types.StatusOpen,
		Priority:  2,
		IssueType: types.TypeMessage,
		Sender:    "mayor",
		Assignee:  "worker",
		Ephemeral: true,
	}
	reply := &types.Issue{
		ID:        "startup-reply",
		Title:     "Startup thread reply",
		Status:    types.StatusOpen,
		Priority:  2,
		IssueType: types.TypeMessage,
		Sender:    "worker",
		Assignee:  "mayor",
		Ephemeral: true,
	}
	wisp := &types.Issue{
		ID:        "startup-wisp",
		Title:     "Startup wisp",
		Status:    types.StatusOpen,
		Priority:  2,
		IssueType: types.TypeTask,
		Ephemeral: true,
	}
	for _, issue := range []*types.Issue{root, reply, wisp} {
		if err := sourceStore.CreateIssue(ctx, issue, "test"); err != nil {
			t.Fatalf("CreateIssue(%s): %v", issue.ID, err)
		}
	}
	if err := sourceStore.AddDependency(ctx, &types.Dependency{
		IssueID: reply.ID, DependsOnID: root.ID, Type: types.DepRepliesTo,
	}, "test"); err != nil {
		t.Fatalf("AddDependency(replies-to): %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceBeadsDir, "redirect"), []byte(targetBeadsDir+"\n"), 0o644); err != nil {
		t.Fatalf("write redirect: %v", err)
	}
	if err := sourceStore.Close(); err != nil {
		t.Fatalf("close source store before reopen: %v", err)
	}

	err = withStorage(ctx, nil, sourceDBPath, func(s storage.DoltStorage) error {
		gotRoot, err := s.GetIssue(ctx, root.ID)
		if err != nil {
			return err
		}
		if gotRoot == nil || gotRoot.ID != root.ID || !gotRoot.Ephemeral {
			return fmt.Errorf("root message missing after reopen: %#v", gotRoot)
		}

		gotReply, err := s.GetIssue(ctx, reply.ID)
		if err != nil {
			return err
		}
		if gotReply == nil || gotReply.ID != reply.ID || !gotReply.Ephemeral {
			return fmt.Errorf("reply message missing after reopen: %#v", gotReply)
		}
		if parent := findRepliesTo(ctx, reply.ID, s); parent != root.ID {
			return fmt.Errorf("reply parent = %q, want %q", parent, root.ID)
		}

		gotWisp, err := s.GetIssue(ctx, wisp.ID)
		if err != nil {
			return err
		}
		if gotWisp == nil || gotWisp.ID != wisp.ID || !gotWisp.Ephemeral {
			return fmt.Errorf("wisp missing after reopen: %#v", gotWisp)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("withStorage() reopen failed: %v", err)
	}
	if got := os.Getenv("BEADS_DOLT_SERVER_DATABASE"); got != sourceCfg.GetDoltDatabase() {
		t.Fatalf("BEADS_DOLT_SERVER_DATABASE = %q, want %q", got, sourceCfg.GetDoltDatabase())
	}
}

func TestIssueIDCompletion_UsesMetadataWhenStoreNil(t *testing.T) {
	originalStore := store
	originalDBPath := dbPath
	originalRootCtx := rootCtx
	defer func() {
		store = originalStore
		dbPath = originalDBPath
		rootCtx = originalRootCtx
	}()

	ctx := context.Background()
	rootCtx = ctx

	testDBPath := filepath.Join(t.TempDir(), "dolt")
	testStore := newTestStoreIsolatedDB(t, testDBPath, "cfg")
	if err := testStore.CreateIssue(ctx, &types.Issue{
		ID:        "cfg-abc1",
		Title:     "Completion target",
		Status:    types.StatusOpen,
		Priority:  1,
		IssueType: types.TypeTask,
	}, "test"); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}

	store = nil
	dbPath = testDBPath

	completions, directive := issueIDCompletion(&cobra.Command{}, nil, "cfg-a")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %d, want %d", directive, cobra.ShellCompDirectiveNoFileComp)
	}
	if len(completions) != 1 {
		t.Fatalf("len(completions) = %d, want 1 (%v)", len(completions), completions)
	}
	if len(completions[0]) < len("cfg-abc1") || completions[0][:len("cfg-abc1")] != "cfg-abc1" {
		t.Fatalf("completion = %q, want prefix %q", completions[0], "cfg-abc1")
	}
}

func TestResolveCommandBeadsDir_NoCWDFallbackForExplicitPath(t *testing.T) {
	// Set up project A with metadata so FindBeadsDir() discovers it from CWD.
	projectA := t.TempDir()
	beadsDirA := filepath.Join(projectA, ".beads")
	if err := os.MkdirAll(filepath.Join(beadsDirA, "dolt"), 0o755); err != nil {
		t.Fatalf("mkdir beads dir A: %v", err)
	}
	cfgA := &configfile.Config{
		Database:     "dolt",
		Backend:      configfile.BackendDolt,
		DoltDatabase: "project_a_db",
	}
	if err := cfgA.Save(beadsDirA); err != nil {
		t.Fatalf("save metadata A: %v", err)
	}

	// Project B: .beads/dolt exists but metadata.json is missing.
	// This triggers the bug: filepath.Dir(dbPath) gives the correct
	// .beads dir but configfile.Load returns nil, so the old code falls
	// through to FindBeadsDir() which discovers project A instead.
	projectB := t.TempDir()
	beadsDirB := filepath.Join(projectB, ".beads")
	if err := os.MkdirAll(filepath.Join(beadsDirB, "dolt"), 0o755); err != nil {
		t.Fatalf("mkdir beads dir B: %v", err)
	}

	// CWD is inside project A so FindBeadsDir() discovers A
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectA); err != nil {
		t.Fatalf("chdir to project A: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// Simulate --db pointing to project B's database path
	dbPathB := filepath.Join(beadsDirB, "dolt")
	got := resolveCommandBeadsDir(dbPathB)

	// Must resolve to project B's .beads, NOT project A's.
	// The old code falls back to FindBeadsDir() and returns beadsDirA.
	if !utils.PathsEqual(got, beadsDirB) {
		t.Fatalf("resolveCommandBeadsDir(%q) = %q, want %q", dbPathB, got, beadsDirB)
	}
}

func TestGetGitHubConfigValue_UsesMetadataWhenStoreNil(t *testing.T) {
	// github.token is now a YAML-only key (not stored in Dolt DB) to avoid
	// leaking secrets when pushing to remotes. Test that the env-var fallback
	// still works when the store is nil.
	originalStore := store
	originalDBPath := dbPath
	defer func() {
		store = originalStore
		dbPath = originalDBPath
	}()

	ctx := context.Background()
	store = nil
	dbPath = ""

	t.Setenv("GITHUB_TOKEN", "ghp_test_token")

	if got := getGitHubConfigValue(ctx, "github.token"); got != "ghp_test_token" {
		t.Fatalf("getGitHubConfigValue() = %q, want %q", got, "ghp_test_token")
	}
}
