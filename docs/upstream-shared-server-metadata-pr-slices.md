# Upstream Shared-Server Metadata PR Slices

Date: 2026-04-17

## Baseline

- Current upstream branch under review: `gastownhall/beads#3242` (`628cf506`, branch `bella-be-drp-pr`).
- Comparison baseline for this review: `origin/main` at `99b2d2ff`.
- The acceptance-matrix follow-up from `be-2az` has already landed in this fork, so the packaging work here is about extracting the still-valid upstreamable pieces from `#3242`, not replaying the old branch wholesale.

## Existing Upstream Context

- Open PR: `#3242` `fix: repair shared-server bootstrap and doctor metadata drift`
- Relevant closed issues: `#2922` (shared remote `project_id` overwrite), `#2590` (configured port ignored), `#2627` (helper reopen paths lose `dolt_database`), `#1834` (metadata drift bucket)
- Related open issue: `#2765` (`test: SQL edge cases, doctor recovery, and chaos testing for Dolt backend`) for any expanded server-recovery coverage that does not fit a minimal bug-fix PR
- Relevant merged PRs on current `main`: `#2947` (treat shared-server as server mode), `#3016` (bootstrap respects shared-server database path), `#3270` (global `beads_global` database support)

## Packaging Rule

- Do not open parallel draft PRs for this topic while `#3242` is still open.
- Preferred path: update `#3242` into the first narrow functional slice, then open at most one follow-up PR after the first slice is either merged or explicitly superseded.
- If maintainers prefer a fresh PR instead of force-updating `#3242`, close `#3242` first and reference it as superseded by the narrower replacement.

## Slice Mapping

| Slice | Upstream mapping | Action |
| --- | --- | --- |
| Slice 1: doctor metadata reconciliation core | PR `#3242`; issues `#2922`, `#2627`, `#2372` | Rework `#3242` into this slice first. This is the strongest standalone bug fix. |
| Slice 2: bootstrap preflight metadata repair | PR `#3242`; issue `#2922`; depends on current `main` state after `#3016`/`be-2az`-equivalent logic | Only open after Slice 1 is rebased and narrowed. Keep it additive to current bootstrap reconciliation. |
| Slice 3: UX-only recovery messaging | PR `#3242`; no dedicated issue needed | Fold into Slice 1 if tiny, or send as a separate cleanup PR only if reviewers ask for it. |
| Extra coverage beyond minimal regressions | issue `#2765` | Do not widen bug-fix slices just to satisfy broader chaos/server test ambitions. |

## Slice Plan

### Slice 1: doctor metadata reconciliation core

Scope:
- `cmd/bd/doctor/fix/metadata.go`
- `cmd/bd/doctor/fix/metadata_test.go`

Keep:
- `ResolveAuthoritativeServerMetadata`
- shared-server/server-mode catalog probing
- `project_id`-based `dolt_database` repair
- configured-database `project_id` backfill/repair
- ambiguity detection when multiple databases match the same `project_id`

Why this slice stands alone:
- It is the core repair primitive the rest of the PR depends on.
- It can be reviewed as a bounded server-mode metadata repair without mixing bootstrap control flow or CLI startup churn.
- It maps cleanly back to prior user-facing regressions around shared-server identity drift and wrong-database reopen behavior.

Dolt compatibility notes:
- Restrict to server/shared-server mode only.
- Probe via `SHOW DATABASES`, then read `issues`/`metadata` in each candidate database.
- Fail closed on ambiguous `project_id` matches; do not guess across unrelated databases.

### Slice 2: bootstrap preflight metadata repair

Scope:
- `cmd/bd/bootstrap.go`
- `cmd/bd/bootstrap_test.go`

Keep:
- the `applyBootstrapMetadataRepair` hook
- wiring bootstrap startup through the authoritative metadata resolver before planning/executing bootstrap
- tests that prove bootstrap adopts the repaired config before continuing

Do not carry forward from `#3242`:
- removals of `finalizeSyncedBootstrap` post-sync identity adoption
- removals of the bootstrap reopen/warmup path
- older helper deletions that predate the `be-2az` acceptance-matrix repair work now present on `main`

Why this slice depends on Slice 1:
- It is only plumbing if the doctor-side repair primitive does not already exist.
- Its rationale is narrower if Slice 1 already establishes the authoritative-metadata repair model upstream.

Dolt compatibility notes:
- Rebase this slice onto current `main` and preserve the newer post-sync reconciliation path.
- The upstream change should be additive: repair metadata before bootstrap acts, not undo newer bootstrap identity fixes.

### Slice 3: UX-only recovery messaging

Scope:
- the recovery hint in `cmd/bd/main.go`

Keep:
- the extra mismatch guidance telling users to try `bd doctor --fix` or `bd bootstrap`

Why this should stay tiny:
- The message is useful regardless of whether the heavier metadata-repair code lands in the same batch.
- It avoids bundling unrelated `main.go` behavior changes into the functional slices.

## Changes To Drop From The Old Branch

These hunks from `#3242` are not good upstream slices against current `main`:

- `cmd/bd/main.go` removal of `--global`
  Reason: stale versus merged PR `#3270`, which added global shared-server support.

- `cmd/bd/main.go` redirect override refactor
  Reason: too broad for the metadata-drift bug, and it changes no-DB command routing behavior at the same time.

- `cmd/bd/doctor/database.go` + `FixMissingMetadata` switch from `GetLocalMetadata`/`SetLocalMetadata` to durable metadata for `bd_version`
  Reason: this is a storage-semantics change, not a metadata-drift fix. It should only move upstream as its own issue/PR if there is a separate problem statement.

- `cmd/bd/main.go` tip timestamp write change from `SetLocalMetadata` to `SetMetadata`
  Reason: same persistence-semantics concern; unrelated to shared-server metadata recovery.

## Review Notes

- `#3242` currently has no human review signal; only Codecov feedback is present.
- The biggest review risk is that the open branch mixes a real bug fix with stale rebased-away logic from older `main` snapshots.
- The safest upstream path is three small PRs: doctor repair primitive, bootstrap wiring, and optional UX text.
- If a maintainer wants just one functional PR, combine Slice 1 and Slice 2 only after rebasing onto current `main`; do not include the dropped hunks above.
- Avoid duplicate draft slop by treating `#3242` as the single active review thread until it is either narrowed or replaced.

## Outcome

- Upstreamable now: Slice 1, Slice 2, Slice 3.
- Needs fresh rebase before opening: Slice 2.
- Should be split out or abandoned: the `main.go` flag/routing churn and the durable-`bd_version` changes from the original `#3242` branch.
