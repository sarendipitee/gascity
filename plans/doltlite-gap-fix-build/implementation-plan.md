---
schema: gc.build.plan.v1
workflow:
  id: gc-kg86
  formula: jj-build
methodology:
  pack: gascity-jj-base
  name: jj-build
producer:
  formula: jj-build
  stage: plan
  attempt: 1
plan_slug: doltlite-gap-fix-build
phase: implementation-plan
rig: gascity
rig_root: /data/projects/doltlite-gascity/gascity
artifact_root: /data/projects/doltlite-gascity/gascity/plans/doltlite-gap-fix-build
status: approved
created_at: 2026-06-28T12:00:35Z
updated_at: 2026-06-28T12:00:35Z
trace:
  upstream:
    - path: plans/doltlite-gap-fix-build/fix-handoff.md
      hash: sha256:881fdbcf8fbcd9d03d3c72c99a3da41782b6c19714637c2218f9fe5e254bc93e
    - path: plans/doltlite-gap-fix-build/beads-doltlite-gap-summary.md
      hash: sha256:130ef6071ee09803f37d70c33e40473c699d67cdeb929fa0efcabf1c7db9f0d8
    - path: plans/doltlite-vcs-mirroring-audit/gap-analysis-report.md
      hash: sha256:e4349ece086e1c5927b110ba5acfef68de27b7476193a740d5fda1ece3152a6a
    - path: /data/projects/doltlite-gascity/doltlite-gap-fix-context.yaml
      hash: sha256:2948252a6da6434a3ced5d9064b41610b8e410f5ae4ece2fc87ed490d512152c
  coverage:
    - id: GAP-001
      status: covered
    - id: GAP-002
      status: covered
    - id: GAP-003
      status: covered
    - id: GAP-004
      status: covered
    - id: GAP-005
      status: covered
    - id: GAP-006
      status: covered
    - id: GAP-007
      status: covered
    - id: GAP-008
      status: covered
    - id: GAP-009
      status: covered
---

# Implementation Plan: DoltLite Gap Fix Build

## Summary

Implement the DoltLite integration gap fixes as a staged convoy that keeps live
beads in DoltLite and keeps workflow documents in the `default@` jj artifact
root. The work should close the Beads-DoltLite storage safety gaps, prove the
linked DoltLite engine behavior, make the reusable pack boundary explicit,
restore Gas City workflow/status parity for DoltLite-backed cities, add
contention coverage, and produce a final report from item summaries.

The build must preserve the serverless DoltLite contract: no Dolt SQL server,
runtime port, or `.gc/runtime/packs/dolt/dolt-state.json` dependency for
DoltLite cities. It must not build, install, or replace live `gc`, `bd`, or
`doltlite-client` binaries, and it must not restart the live city or supervisor
without explicit operator approval.

## Current System

The source evidence identifies two completed gap-analysis inputs:

- Beads-DoltLite backend gaps from workflow `gc-fc8y`, summarized in
  `plans/doltlite-gap-fix-build/beads-doltlite-gap-summary.md`.
- Gas City DoltLite VCS mirroring gaps from workflow `gc-qwr0`, recorded in
  `plans/doltlite-vcs-mirroring-audit/gap-analysis-report.md`.

The current Gas City checkout already has DoltLite provider and diagnostic
surfaces in `examples/beads-doltlite/`, `tools/doltlite-client/`,
`cmd/gc/beads_provider_lifecycle.go`, `cmd/gc/store_health.go`,
`internal/storehealth/storehealth.go`, `internal/api/store_health.go`, and
`internal/beads/doltlite_read_store.go`. The reusable pack and maintenance
commands live under `examples/beads-doltlite/`, while generic workflow and VCS
behavior is currently spread across Gas City and Gas Town packs rather than a
complete DoltLite-ready branch/refinery contract.

The Beads-side report names storage risks that must be handled in the
Beads-DoltLite checkout when available: raw `bd sql` writes can bypass
provider-owned write safety, `is_blocked` can go stale if dependency/status
mutations do not maintain it consistently, and multiprocess live-city
contention is not proved by focused tests. The Gas City report adds workflow
parity gaps: the DoltLite route lacks the proven branch/refinery handoff
contract, store health/status remains too Dolt-specific in places, and the
`examples/beads-doltlite` pack is not clearly provider-only or workflow-capable.

## Proposed Implementation

### Storage Safety

Fix Beads-DoltLite mutation paths first, because later integration tests depend
on reliable writes. In the Beads/DoltLite codebase, audit `cmd/bd/sql.go`, the
DoltLite accessor and mutation helpers, external write locking, stale
connection retry/reset behavior, typed mutation hooks, `CommitPending`, and all
dependency/status mutation paths that can affect blocked state.

Raw SQL writes against a DoltLite store should either route through the
provider-owned write path or fail with a clear unsupported-operation error. If
`is_blocked` remains part of the schema, every mutation that can change blocked
state must maintain it; otherwise read paths should stop depending on stale
stored state. Cover dependency add, dependency close, issue close/reopen, defer,
delete, and parent-child paths with focused tests. Write the item summary to
`plans/doltlite-gap-fix-build/implementation-items/storage-safety-summary.md`.

### Linked Engine Evidence

Make linked-engine diagnostics prove the actual linked DoltLite path instead of
only proving fallback or process bootstrap behavior. Update
`examples/beads-doltlite/commands/build/run.sh`,
`examples/beads-doltlite/health_command_test.go`,
`examples/beads-doltlite/template-fragments/city-understanding.template.md`,
and `tools/doltlite-client/` as needed so diagnostics state whether the binary
is using libdoltlite and whether required DoltLite SQL functions are registered.

Focused tests should exercise or verify `dolt_branch`, `dolt_checkout`,
`dolt_merge`, `dolt_log`, `dolt_diff_*`, `dolt_hashof_*`, and `dolt_gc()` when
the linked engine supports them. Missing functions should fail with a stable
diagnostic that names the function and linked-engine context. Write the item
summary to
`plans/doltlite-gap-fix-build/implementation-items/linked-engine-evidence-summary.md`.

### Pack Maintenance Semantics

Make the `examples/beads-doltlite` pack boundary explicit. Update
`examples/beads-doltlite/pack.toml`,
`examples/beads-doltlite/formulas/mol-doltlite-maintenance.toml`,
`examples/beads-doltlite/commands/flatten/run.sh`, and
`examples/beads-doltlite/commands/gc/run.sh` so the pack is documented and
configured as either provider-only or workflow-capable.

If the pack is workflow-capable, it must import or define the roles and
formulas it references. If it is provider-only, maintenance formulas must avoid
unavailable role names. Flatten/gc commands should emit a stable JSON schema
for success and failure. Non-fatal maintenance should record degraded state,
while fatal maintenance should preserve nonzero exit status. Write the item
summary to
`plans/doltlite-gap-fix-build/implementation-items/pack-maintenance-summary.md`.

### Backend-Aware Status

Update status and health surfaces so DoltLite-backed cities do not inherit
Dolt-only assumptions. Review `cmd/gc/store_health.go`,
`internal/storehealth/storehealth.go`, `internal/api/store_health.go`,
`cmd/gc/beads_provider_lifecycle.go`, and related provider tests. The CLI and
API should report the selected backend, health, and diagnostic state without
requiring a Dolt SQL server, a runtime port, or a Dolt state file for DoltLite.

Focused tests should cover DoltLite city status, API store-health output, and
failure/degraded states from pack maintenance commands. Write the item summary
to `plans/doltlite-gap-fix-build/implementation-items/gascity-status-summary.md`.

### VCS Workflow Parity

Add the smallest reusable branch/refinery contract needed for DoltLite workflow
parity, preferably in pack/formula/prompt assets rather than generic Go logic.
Use the proven Gas Town behavior as the evidence source: deterministic
per-bead branch names, `branch`, `target`, `branch_ready`, and `halt_reason`
metadata, branch creation from the resolved base, current-branch validation,
diff-based false-completion refusal, publish/merge refusal for empty branches,
and rejection/requeue metadata.

This work should wait for linked-engine diagnostics so missing DoltLite SQL
functions are reported consistently. Candidate surfaces include the Gas City
workflow assets under `gascity-packs/gascity/assets/workflows/`, the imported
Gas Town formula and prompt assets, and the DoltLite pack when the boundary
decision requires local overrides. Write the item summary to
`plans/doltlite-gap-fix-build/implementation-items/vcs-parity-summary.md`.

### Contention Integration

After storage safety, linked-engine evidence, and backend-aware status are in
place, add live-city multiprocess contention coverage. The test should exercise
the final DoltLite behavior rather than documenting known pre-fix failures:
concurrent claim/update/close style mutations, stale connection recovery,
commit behavior, store health, and status reporting under contention.

Keep this focused and isolated. Do not run the full Go suite locally, do not
restart the live city, and use explicit test fixtures or temporary cities.
Write the item summary to
`plans/doltlite-gap-fix-build/implementation-items/contention-integration-summary.md`.

### Final Report

Fan in all item summaries and write
`plans/doltlite-gap-fix-build/final-report.md`. The report should map each
GAP-001 through GAP-009 to the implemented change, focused verification,
remaining risk, and any source checkout that was unavailable. It should also
record whether live binary installation, city restart, publication, push, or PR
opening were intentionally skipped.

## Testing

Use focused tests only. Do not run the full test suite locally.

- Beads-DoltLite storage tests for raw SQL rejection/routing, dependency
  mutations, blocked-state consistency, stale connection recovery, and commit
  behavior.
- `examples/beads-doltlite` tests for linked-engine diagnostics, maintenance
  command JSON, flatten/gc failure behavior, pack imports, and store health.
- `tools/doltlite-client` focused tests for linked function availability and
  diagnostics.
- Gas City status/API tests around `cmd/gc/store_health.go`,
  `internal/storehealth/storehealth.go`, and `internal/api/store_health.go`.
- Workflow asset tests for branch metadata, branch readiness, current-branch
  validation, empty-diff refusal, rejection metadata, and publish/merge refusal.
- One isolated live-city or integration test for multiprocess DoltLite
  contention once the lower-level behavior is covered.

If API schema files, `internal/api/`, `cmd/gc/dashboard/`, or generated
dashboard types change, also run the dashboard checks required by the repo
instructions for those paths.

## Rollout

1. Land storage safety, linked-engine evidence, pack maintenance, and
   backend-aware status as parallel implementation items when their source
   checkouts are available.
2. Land VCS workflow parity after linked-engine diagnostics provide stable
   missing-function behavior.
3. Land contention integration after storage, engine, and status changes are in
   place.
4. Produce the final report from item summaries and keep `push=false` and
   `open_pr=false` unless explicitly authorized.

No implementation item should install replacement live binaries, restart the
city, or mutate live supervisor state without operator approval. Missing source
checkouts should be recorded as item failures with exact paths instead of
inventing local substitutes.

## Open Questions

- Is the Beads-DoltLite source checkout available to the implementation workers
  at `/data/projects/doltlite-gascity/beads-doltlite`, or should Beads-side
  fixes be reported as blocked with exact missing paths?
- Should `examples/beads-doltlite` be provider-only or workflow-capable after
  this build? The implementation should decide this from current pack imports
  and formula references, then make the boundary explicit.
- Should the reusable branch/refinery contract live in the generic Gas City
  workflow pack, a shared VCS methodology pack, or a DoltLite-specific pack
  override? Prefer the smallest pack-level slice that avoids hardcoded roles in
  Go.
