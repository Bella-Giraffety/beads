# Bootstrap Reconciliation Audit

Date: 2026-04-17

## Baseline

- This revalidation was performed against `origin/main` at `ff63a1d0`.
- The relevant upstream bootstrap fixes called out by `be-7ip.1` are already present in this fork; the remaining work for this bead is to document the keep/cherry-pick/reimplement/supersede decisions and the intentional lack of code divergence in these areas.

## Decisions

| Area | Relevant commits | Decision | Current evidence |
| --- | --- | --- | --- |
| Shared-server clone-local state | `50f715b1`, `31d51232` | Keep | Clone-local state remains in Dolt-ignored tables and bootstrap ensures those tables exist after clone/server restart. See `cmd/bd/init.go`, `internal/storage/dolt/wisps.go`, and `internal/storage/dolt/store.go`. |
| Bootstrap metadata persistence | `4aa573db`, `9faaf3d7`, `4f7a0d42` | Keep | `bd init` repairs and persists `metadata.json`, writes `config.yaml`, and converges shared-server metadata with database identity and server mode. See `cmd/bd/init.go` and `cmd/bd/main.go`. |
| Issue-prefix persistence | `ebcb901d` | Keep | `bd init` seeds `issue_prefix` only when absent, preserving shared DB state, and config writes remain explicit rather than bundled into generic commits. See `cmd/bd/init.go` and `internal/storage/dolt/store.go`. |
| Shared-server path resolution | `f9ec9859`, `efdb8f8c`, `3b5c4e2f` | Keep | Explicit DB-path resolution maps back to the owning `.beads` directory, preserves routed source database selection, and avoids CWD drift. See `cmd/bd/main.go` and `cmd/bd/store_reopen_test.go`. |
| Worktree-aware formula/config lookup | `9b4896f0`, `785253bc`, `368cce15` | Keep | Formula search paths and config repo-root fallback resolve through the effective beads directory rather than raw CWD assumptions. See `internal/formula/parser.go`, `internal/formula/parser_test.go`, and `cmd/bd/config_worktree_test.go`. |

## Notes By Area

### Shared-server clone-local state

- The clone-local split remains the right behavior for Gas Town because worktree- and machine-local state should not create merge conflicts in shared Dolt history.
- `internal/storage/dolt/wisps.go` still keeps wisps and related tables in `dolt_ignored` storage, and `internal/storage/dolt/store.go` still recreates ignored tables on server-mode startup.
- `cmd/bd/init.go` continues to route shared-server bootstrap through server mode immediately so initialization does not stamp shared-server workspaces with embedded-only state.

Decision: keep as-is. No cherry-pick or reimplementation needed.

### Bootstrap metadata persistence

- `cmd/bd/init.go` persists both runtime metadata (`metadata.json`) and durable project config (`config.yaml`) during initialization.
- The init path now adopts an existing database project identity when present, writes the chosen Dolt database name into metadata, and persists shared-server intent to YAML.
- `cmd/bd/main.go` still contains the shared-server embedded/server mismatch repair path so stale `metadata.json` does not silently hide server-backed state.

Decision: keep. This supersedes older manual metadata repair workarounds.

### Issue-prefix persistence

- The important behavior is still that SQL database naming and issue-prefix naming are separate concerns.
- `cmd/bd/init.go` only seeds `issue_prefix` when the config table does not already contain one, which avoids clobbering shared-server or reused-database setups.
- Generic store commits do not implicitly sweep config changes; callers changing config do so intentionally.

Decision: keep. No fork-only override should reintroduce implicit prefix commits.

### Shared-server path resolution

- `cmd/bd/main.go` resolves explicit `--db` selections back to the owning `.beads` directory instead of falling back to whichever workspace `FindBeadsDir()` sees from the current shell.
- Redirect handling still preserves the source Dolt database across redirects, preventing reads from drifting into the wrong catalog.
- `cmd/bd/store_reopen_test.go` covers the explicit-path/no-CWD-fallback cases that matter for shared-server bootstrap and routed execution.

Decision: keep. There is no remaining cherry-pick in this area.

### Worktree-aware formula/config lookup

- `internal/formula/parser.go` still derives default formula search paths from the resolved beads directory, which is the correct behavior for shared `.beads` state and worktree-heavy usage.
- `internal/formula/parser_test.go` and `cmd/bd/config_worktree_test.go` cover the worktree fallback behavior on both formula lookup and config repo-root detection.
- This is the exact class of fix Gas Town relies on because formulas, hooks, and config are routinely consumed from rig worktrees rather than the main repo root.

Decision: keep. No local divergence is useful here.

## Remaining Intentional Divergence

- No code divergence remains in the audited bootstrap areas.
- Gas Town-specific behavior here is operational rather than fork-specific code: rig-local `.beads` state, hook molecules, and worker workflows depend on these upstream-compatible paths without needing extra patching in the audited files.

## Outcome

- Keep: all audited fixes above.
- Cherry-pick: none pending.
- Reimplement: none needed.
- Supersede: older manual/shared-server bootstrap workarounds are superseded by the current paths already present on `main`.
