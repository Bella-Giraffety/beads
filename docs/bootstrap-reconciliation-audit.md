# Bootstrap Reconciliation Audit

Date: 2026-04-16

## Baseline

- `origin/main` and upstream `https://github.com/steveyegge/beads` both resolved to `4e3dbb6d` when this audit was performed.
- Result: there are no pending upstream cherry-picks for the bootstrap areas scoped by `be-7ip.1`; the work is to verify that the right fixes are present and document the remaining intentional divergence.

## Decisions

| Area | Relevant commits | Decision | Current evidence |
| --- | --- | --- | --- |
| Shared-server clone-local state | `50f715b1`, `31d51232` | Keep | Clone-local state is stored in Dolt-ignored tables and bootstrap ensures those tables exist after clone. See `cmd/bd/init.go`, `internal/storage/dolt/wisps.go`, `internal/storage/dolt/store.go`, and `cmd/bd/bootstrap.go`. |
| Bootstrap metadata persistence | `4aa573db`, `9faaf3d7`, `4f7a0d42` | Keep | `bd init` writes and repairs `metadata.json`, persists `config.yaml`, and converges shared-server metadata with database identity. See `cmd/bd/init.go` and `cmd/bd/main.go`. |
| Issue-prefix persistence | `ebcb901d` | Keep | `bd init` seeds `issue_prefix` into the config table without clobbering existing shared DB state, and the commit path keeps config writes explicit. See `cmd/bd/init.go` and `internal/storage/dolt/store.go`. |
| Shared-server path resolution | `f9ec9859`, `efdb8f8c`, `3b5c4e2f` | Keep | Explicit DB-path resolution maps back to the owning `.beads` directory, preserves routed server metadata, and avoids CWD fallback drift. See `cmd/bd/main.go` and `cmd/bd/store_reopen_test.go`. |
| Worktree-aware formula/config lookup | `9b4896f0`, `785253bc`, `368cce15` | Keep | Formula search paths and config repo-root fallback resolve through the effective beads directory instead of raw CWD assumptions. See `internal/formula/parser.go`, `internal/formula/parser_test.go`, and `cmd/bd/config_worktree_test.go`. |

## Notes By Area

### Shared-server clone-local state

- The clone-local split is the right long-term choice for Gas Town usage because it prevents worktree- and machine-specific state from generating merge conflicts in shared Dolt history.
- `cmd/bd/init.go` writes `bd_version` as local metadata, while `internal/storage/dolt/wisps.go` and the ignored-table helpers keep ephemeral/wisp state out of versioned history.
- `31d51232` remains relevant because cold-start bootstrap must recreate the ignored tables after clone before higher-level commands touch them.

Decision: keep as-is. No reimplementation needed.

### Bootstrap metadata persistence

- `cmd/bd/init.go` now persists both runtime metadata (`metadata.json`) and durable project config (`config.yaml`) during initialization.
- `cmd/bd/main.go` includes the shared-server embedded/server mismatch repair path so stale metadata does not silently hide server-backed state.
- The later convergence fix (`4f7a0d42`) is the important follow-up for the bootstrap effort here: project identity, selected database, and shared-server mode must agree after bootstrap, not just after a clean init.

Decision: keep. This supersedes any older local bootstrap workaround that relied on manual metadata repair.

### Issue-prefix persistence

- The important behavior is that the SQL database name and the issue prefix are no longer treated as the same thing.
- `cmd/bd/init.go` preserves an existing shared DB prefix and only seeds `issue_prefix` when it is absent.
- `internal/storage/dolt/store.go` still excludes `config` from generic commits, which avoids sweeping up unrelated stale prefix changes; callers that intentionally change config use the explicit config-aware commit path.

Decision: keep. No fork-only override should reintroduce implicit prefix commits.

### Shared-server path resolution

- `cmd/bd/main.go` now resolves explicit `--db` and no-DB command paths back to the owning workspace instead of falling back to whatever `FindBeadsDir()` sees from the current shell directory.
- Redirected/routed execution also preserves the source database name across redirects so shared-server reads do not drift into the wrong catalog.
- The bounded repo-root fallback protects worktree config commands from walking too far and latching onto an unrelated parent repo.

Decision: keep. There is no remaining cherry-pick in this area.

### Worktree-aware formula/config lookup

- `internal/formula/parser.go` uses the resolved beads directory for default formula search paths, which is the right behavior for shared `.beads` state and bare-parent worktrees.
- `cmd/bd/config_worktree_test.go` covers the config-side repo-root fallback and pollution-check behavior.
- This is the exact class of fix Gas Town needs because formulas, hooks, and config are consumed from worktrees far more often than from the main repo root.

Decision: keep. No local divergence is useful here.

## Remaining Intentional Divergence

- No code divergence remains in the audited bootstrap areas; the fork is aligned with upstream at the audited baseline.
- Gas Town-specific behavior is operational rather than code-level here: rig-local `.beads` data, hook molecules, and worker workflows depend on these upstream-compatible codepaths but do not require a separate fork patch in the audited files.

## Outcome

- Keep: all audited fixes above.
- Cherry-pick: none pending.
- Reimplement: none needed.
- Supersede: older manual/shared-server workarounds are superseded by the current upstream-compatible paths now present on `main`.
