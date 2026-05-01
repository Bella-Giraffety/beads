package main

import (
	"context"

	"github.com/steveyegge/beads/internal/configfile"
)

type workspaceIdentityMetadataStore interface {
	GetMetadata(ctx context.Context, key string) (string, error)
}

type workspaceIdentityStatus struct {
	LocalID    string
	DatabaseID string
	Mismatch   bool
}

func (s workspaceIdentityStatus) infoStatus() (string, string, bool) {
	if s.LocalID == "" {
		return "", "", false
	}

	if s.Mismatch {
		return "mismatch", "Workspace metadata.json and database _project_id disagree; suppressing issue-derived diagnostics", true
	}

	if s.DatabaseID == "" {
		return "unverified", "Database _project_id is missing; suppressing issue-derived diagnostics until identity is proven", true
	}

	return "ok", "Workspace and database identities match", true
}

func (s workspaceIdentityStatus) allowsIssueDiagnostics() bool {
	if s.LocalID == "" {
		return true
	}

	return s.DatabaseID != "" && !s.Mismatch
}

func currentWorkspaceIdentity(ctx context.Context, beadsDir string, s workspaceIdentityMetadataStore) workspaceIdentityStatus {
	if s == nil {
		return workspaceIdentityStatus{}
	}

	cfg, err := configfile.Load(beadsDir)
	if err != nil || cfg == nil || cfg.ProjectID == "" {
		return workspaceIdentityStatus{}
	}

	status := workspaceIdentityStatus{LocalID: cfg.ProjectID}
	dbProjectID, err := s.GetMetadata(ctx, "_project_id")
	if err != nil || dbProjectID == "" {
		return status
	}

	status.DatabaseID = dbProjectID
	status.Mismatch = cfg.ProjectID != dbProjectID
	return status
}
