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
