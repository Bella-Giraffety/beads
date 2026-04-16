package dolt

import (
	"context"
	"testing"

	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/types"
)

func TestIsIgnoredTableCorruptionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "read many values checksum",
			err:  errString("writeCommitParentClosure: ReadManyValues: checksum error"),
			want: true,
		},
		{
			name: "plain checksum is ignored",
			err:  errString("checksum error"),
			want: false,
		},
		{
			name: "non corruption error",
			err:  errString("table not found"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIgnoredTableCorruptionError(tt.err); got != tt.want {
				t.Fatalf("isIgnoredTableCorruptionError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunInTransaction_RetriesAfterIgnoredTableRepair(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	if err := store.SetConfig(ctx, "issue_prefix", "test"); err != nil {
		t.Fatalf("SetConfig(issue_prefix): %v", err)
	}

	root := &types.Issue{
		Title:     "Ephemeral molecule",
		IssueType: types.IssueType("molecule"),
		Ephemeral: true,
	}
	step := &types.Issue{
		Title:     "Current step",
		IssueType: types.TypeTask,
		Status:    types.StatusInProgress,
		Ephemeral: true,
	}

	repairInjected := 0
	store.ignoredTableRepairHook = func() error {
		if repairInjected > 0 {
			return nil
		}
		repairInjected++
		return errString("writeCommitParentClosure: ReadManyValues: checksum error")
	}
	defer func() { store.ignoredTableRepairHook = nil }()

	err := store.RunInTransaction(ctx, "test: create ephemeral molecule", func(tx storage.Transaction) error {
		if err := tx.CreateIssue(ctx, root, "test"); err != nil {
			return err
		}
		if err := tx.CreateIssue(ctx, step, "test"); err != nil {
			return err
		}
		return tx.AddDependency(ctx, &types.Dependency{
			IssueID:     step.ID,
			DependsOnID: root.ID,
			Type:        types.DepParentChild,
		}, "test")
	})
	if err != nil {
		t.Fatalf("RunInTransaction() after repair: %v", err)
	}
	if repairInjected != 1 {
		t.Fatalf("expected one injected corruption, got %d", repairInjected)
	}

	progress, err := store.GetMoleculeProgress(ctx, root.ID)
	if err != nil {
		t.Fatalf("GetMoleculeProgress() after repair: %v", err)
	}
	if progress.Total != 1 {
		t.Fatalf("progress.Total = %d, want 1", progress.Total)
	}
	if progress.InProgress != 1 {
		t.Fatalf("progress.InProgress = %d, want 1", progress.InProgress)
	}
	if progress.CurrentStepID != step.ID {
		t.Fatalf("progress.CurrentStepID = %q, want %q", progress.CurrentStepID, step.ID)
	}

	issue, err := store.GetIssue(ctx, step.ID)
	if err != nil {
		t.Fatalf("GetIssue(step) after repair: %v", err)
	}
	if issue == nil || issue.ID != step.ID {
		t.Fatalf("GetIssue(step) returned %+v", issue)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
