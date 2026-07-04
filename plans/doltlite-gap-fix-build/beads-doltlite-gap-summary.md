# Beads-DoltLite Backend Gap Summary

This summary is derived from:

`/data/projects/doltlite-gascity/beads-doltlite/plans/doltlite-backend-upstream-audit/gap-analysis-report.md`

The source workflow root was `gc-fc8y`, and the report step was `gc-lkvt`.

## Verdict

Fail for linked-binary rollout readiness.

The report found good parity work around DoltLite raw DB access, commit policy,
stale-connection retry/reset, and context diagnostics outside git workspaces.
It still blocks linked `gc`, `bd`, and `doltlite-client` rollout because raw SQL
writes, `is_blocked` schema integration, linked-engine evidence, and live
contention coverage are incomplete.

## Findings To Fix

1. `cmd/bd/sql.go` runs write SQL through `accessor.UnderlyingDB()` with
   direct `db.ExecContext`. For DoltLite this bypasses `withDBWrite`, the
   external write lock, stale-connection retry/reset behavior, typed mutation
   hooks, and `CommitPending`.
2. `is_blocked` exists only in migrations. Dynamic blocked-ID computation is
   still the real source of truth. If a fast path reads `is_blocked`, it can
   become stale across dependency, status, wisp, parent-child, close, reopen,
   defer, and delete operations.
3. Focused DoltLite tests did not prove the linked engine path because the
   runtime lacked DoltLite SQL functions such as `dolt_checkout`.
4. Current concurrency evidence is too narrow. It covers stale handles but not
   multiprocess live-city claim/update/close/create/ready traffic.
5. The Beads-side repo cannot audit `cmd/gc` integration. Gas City-side store
   opening, hook claiming, session reconciliation, ready/assigned observation,
   and linked install flow need companion coverage.
6. Upstream freshness was not proven because the report-only workflow did not
   fetch and compare against a fresh upstream baseline.

## Recommended Implementation Direction

- Block or safely implement DoltLite raw `bd sql` writes before rollout.
- Decide whether `is_blocked` is future-only or production-maintained, then
  either remove/defer the migrations or add maintenance and parity tests.
- Add a linked-engine bootstrap diagnostic and rerun focused DoltLite tests
  only when the linked functions are actually registered.
- Add a multiprocess contention test using real `bd` or `doltlite-client`
  processes against the same DoltLite database.
- Fold the Gas City companion audit results into implementation boundaries so
  storage and orchestration fixes are reviewed together.
