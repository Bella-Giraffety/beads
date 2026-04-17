# Bootstrap Reconciliation Audit

Date: 2026-04-17

## Baseline

- `origin/main` resolved to `ff63a1d0` during this audit.
- The preserved be-7ip.1 recovery commit (`e5b8da5f`) captured the same conclusion earlier on 2026-04-16, but that documentation was not present on the current jasper worktree.
- Result: there are no pending upstream cherry-picks for the bootstrap areas scoped by `be-7ip.1`; the work is to verify that the relevant hardening is already present and document the remaining intentional divergence.

## Decisions

| Area | Relevant commits | Decision | Current evidence |
| --- | --- | --- | --- |
| Shared-server clone-local state | `50f715b1`, `31d51232` | Keep | Clone-local state is stored in Dolt-ignored tables and bootstrap recreates ignored tables on open. See `internal/storage/dolt/wisps.go` and `internal/storage/dolt/store.go`. |
| Bootstrap metadata persistence | `4aa573db`, `9faaf3d7`, `4f7a0d42` | Keep | `bd init` persists metadata/config state and converges project identity, database selection, and server mode. See `cmd/bd/init.go` and `cmd/bd/main.go`. |
| Issue-prefix persistence | `ebcb901d` | Keep | `bd init` only seeds `issue_prefix` when absent, and generic Dolt commits exclude `config` so stale prefix changes are not swept in. See `cmd/bd/init.go` and `internal/storage/dolt/store.go`. |
| Shared-server path resolution | `f9ec9859`, `efdb8f8c`, `3b5c4e2f` | Keep | Explicit DB-path resolution maps back to the owning `.beads` directory and preserves redirected source database metadata. See `cmd/bd/main.go`. |
| Worktree-aware formula/config lookup | `9b4896f0`, `785253bc`, `368cce15` | Keep | Formula search paths resolve through the effective beads directory, and worktree fallback coverage exists for config lookup. See `internal/formula/parser.go` and `cmd/bd/config_worktree_test.go`. |

## Notes By Area

### Shared-server clone-local state

- Wisp and auxiliary tables live in Dolt-ignored tables so machine-local/runtime state does not pollute shared history.
- `internal/storage/dolt/store.go` recreates ignored tables during store open because ignored tables do not persist across restarts/branches.

Decision: keep as-is. No reimplementation needed.

### Bootstrap metadata persistence

- `cmd/bd/init.go` persists `metadata.json`, writes durable config state, and adopts database authority for project identity when reopening an existing/shared database.
- `cmd/bd/main.go` repairs the stale `dolt_mode=embedded` case when shared-server mode is active so server-backed state is not hidden by old metadata.

Decision: keep. This supersedes older manual metadata-repair workflows.

### Issue-prefix persistence

- `cmd/bd/init.go` avoids clobbering an existing shared-database `issue_prefix`; it only seeds the prefix when missing.
- `internal/storage/dolt/store.go` excludes `config` from generic commits and requires explicit config-aware commits for intentional config changes.

Decision: keep. No fork-only override should reintroduce implicit config commits.

### Shared-server path resolution

- `cmd/bd/main.go` resolves explicit DB paths back to the owning workspace instead of trusting raw CWD discovery.
- Redirect handling preserves the source database across redirects so routed shared-server commands stay attached to the intended catalog.

Decision: keep. No pending cherry-pick remains in this area.

### Worktree-aware formula/config lookup

- `internal/formula/parser.go` builds default search paths from the resolved beads directory rather than raw cwd assumptions.
- `cmd/bd/config_worktree_test.go` covers worktree fallback to the main repo root when `.beads` lives outside the linked worktree.

Decision: keep. No local divergence is useful here.

## Remaining Intentional Divergence

- No code divergence remains in the audited bootstrap areas.
- Gas Town-specific behavior in this scope is operational rather than fork-specific code: rig-local `.beads` state, hook molecules, and worker workflows ride on these upstream-compatible paths.

## Outcome

- Keep: all audited fixes above.
- Cherry-pick: none pending.
- Reimplement: none needed.
- Supersede: older manual/shared-server workarounds are superseded by the current upstream-compatible paths already present on `main`.
