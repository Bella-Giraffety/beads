package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steveyegge/beads/internal/configfile"
	"github.com/steveyegge/beads/internal/doltdboverride"
)

func TestServerModeIntegrityManualRecoveryDetailUsesScopedDatabaseOverride(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatalf("mkdir beads dir: %v", err)
	}
	if err := (&configfile.Config{
		Backend:      configfile.BackendDolt,
		DoltMode:     configfile.DoltModeServer,
		DoltDatabase: "shared_db",
	}).Save(beadsDir); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	restore := doltdboverride.Push("source_db")
	defer restore()

	detail := serverModeIntegrityManualRecoveryDetail(beadsDir)
	if !strings.Contains(detail, "source_db") {
		t.Fatalf("detail = %q, want scoped source database", detail)
	}
	if strings.Contains(detail, "shared_db") {
		t.Fatalf("detail = %q, should not use redirect target database", detail)
	}
}
