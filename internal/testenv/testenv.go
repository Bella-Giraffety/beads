package testenv

import "os"

// UnsetAmbientDoltEnv removes polecat/worktree Dolt env vars so package tests
// start from a repo-local baseline and opt into overrides explicitly.
func UnsetAmbientDoltEnv() {
	for _, key := range []string{
		"BEADS_DOLT_AUTO_START",
		"BEADS_DOLT_DATA_DIR",
		"BEADS_DOLT_HOST",
		"BEADS_DOLT_PASSWORD",
		"BEADS_DOLT_PORT",
		"BEADS_DOLT_REMOTESAPI_PORT",
		"BEADS_DOLT_SERVER_DATABASE",
		"BEADS_DOLT_SERVER_HOST",
		"BEADS_DOLT_SERVER_MODE",
		"BEADS_DOLT_SERVER_PORT",
		"BEADS_DOLT_SERVER_TLS",
		"BEADS_DOLT_SERVER_USER",
		"BEADS_DOLT_SHARED_SERVER",
	} {
		_ = os.Unsetenv(key)
	}
}
