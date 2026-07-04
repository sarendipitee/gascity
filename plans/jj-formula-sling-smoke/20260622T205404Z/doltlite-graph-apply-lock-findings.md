---
run_id: 20260622T205404Z
phase: sling-smoke-findings
rig: gascity
backend: doltlite
status: draft
---

# DoltLite Graph Apply Lock Findings

## Smoke Commands

Two cataloged JJ formulas were attached through the real pack entry point:

- `jj-build` on source bead `gc-naoa`
- `jj-review` on source bead `gc-9qpx`

Both used:

```bash
gc sling gascity/gc.run-operator <source-bead> --on <formula> --json --nudge
```

Both failed with:

```text
instantiating formula "<formula>" on <source-bead>: bd create --graph: exit status 1: {
  "error": "graph create: doltlite add dependencies: database is locked",
  "schema_version": 1
}
```

## Database Evidence

Despite the CLI failures, the DoltLite database contained complete graph rows.

- `jj-build` root `gc-d5bf`: 31 workflow rows, 43 internal `blocks` deps, 30 internal `tracks` deps
- `jj-review` root `gc-fbxq`: 7 workflow rows, 8 internal `blocks` deps, 6 internal `tracks` deps

The smoke-created beads were closed after inspection with:

- `gc.outcome=fail`
- `gc.failure_class=smoke-test`
- `gc.failure_reason=formula_sling_doltlite_lock_probe`

## Exact Failure Path

Gas City formula graph apply is enabled, but the store path used by the running command is subprocess-backed:

1. `gc sling`
2. `gascity/internal/molecule/graph_apply.go:53` `instantiateViaGraphApply`
3. `gascity/internal/beads/bdstore_graph_apply.go:19` `BdStore.ApplyGraphPlanWithStorage`
4. `gascity/internal/beads/bdstore_graph_apply.go:53` builds `bd create --graph <tmp> --json`
5. `beads-doltlite/cmd/bd/graph_apply.go:183` runs `store.RunInTransaction`
6. `beads-doltlite/internal/storage/doltlite/transaction.go:20` commits SQL changes, then calls `s.Commit`
7. `beads-doltlite/internal/storage/doltlite/version_control.go:63` stages dirty tables with `SELECT dolt_add(?)`
8. `beads-doltlite/internal/storage/doltlite/version_control.go:65` emits `doltlite add dependencies: <err>`

So the observed lock is at DoltLite version staging of the `dependencies` table, after the SQL graph transaction has already committed enough state for the complete graph to be visible.

## Wiring Gaps

- `bd create --graph` has no DoltLite-specialized graph apply implementation. It uses generic per-node and per-edge transaction APIs.
- `DoltliteStore.RunInTransaction` has no way to return "SQL transaction committed, but version commit/staging failed" with enough structured information for `bd create --graph` to return the created ID map.
- `DoltliteStore.CreateIssuesWithFullOptions` loops per issue for non-transactional batch create, so ordinary bulk create is also not truly batched on DoltLite.
- Gas City's DoltLite read fast path does not override graph apply. `DoltliteReadStore` embeds `BdStore`, so `ApplyGraphPlan` falls back to shelling out through `bd create --graph`.

## Required Updates

1. In `beads-doltlite`, teach graph apply / DoltLite transaction code to distinguish body failure from post-SQL version-staging failure.
2. For graph apply, if SQL rows committed but `dolt_add` or `dolt_commit` hit a retryable lock, either retry only the version staging step or return the created ID map with a recoverable commit/staging warning.
3. Add a DoltLite graph-apply regression test where `dolt_add('dependencies')` returns `database is locked` after rows are committed, and assert graph rows are not duplicated on retry.
4. In Gas City, add or wire a DoltLite-native graph apply path instead of inheriting `BdStore.ApplyGraphPlan` through `DoltliteReadStore`, or keep the graph-v2 recovery path mandatory for subprocess-backed DoltLite stores.
