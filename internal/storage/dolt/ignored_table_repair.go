package dolt

import (
	"context"
	"fmt"
	"strings"

	"github.com/steveyegge/beads/internal/storage/schema"
)

func isIgnoredTableCorruptionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "checksum error") {
		return false
	}
	return strings.Contains(errStr, "readmanyvalues") || strings.Contains(errStr, "writecommitparentclosure")
}

func (s *DoltStore) withIgnoredTableRepair(ctx context.Context, op func() error) error {
	err := op()
	if !isIgnoredTableCorruptionError(err) {
		return err
	}
	if repairErr := s.repairIgnoredTables(ctx); repairErr != nil {
		return fmt.Errorf("repair ignored tables after checksum corruption: %w (original error: %v)", repairErr, err)
	}
	return op()
}

func (s *DoltStore) repairIgnoredTables(ctx context.Context) error {
	s.ignoredTableRepairMu.Lock()
	defer s.ignoredTableRepairMu.Unlock()

	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for ignored-table repair: %w", err)
	}
	defer conn.Close()

	if err := schema.RepairIgnoredTables(ctx, conn); err != nil {
		return fmt.Errorf("reset ignored tables: %w", err)
	}
	return nil
}
