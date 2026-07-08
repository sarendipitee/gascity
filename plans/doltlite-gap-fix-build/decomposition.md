---
schema: gc.build.decomposition.v1
workflow:
  id: gc-kg86
  formula: jj-build
methodology:
  pack: gascity-jj-base
  name: jj-build
producer:
  formula: jj-build
  stage: decompose
  attempt: 1
status: approved
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

# Decomposition: DoltLite Gap Fix Build

## Summary

This decomposition turns the approved DoltLite gap-fix handoff into one
implementation convoy. The work is split into Beads storage safety, linked
DoltLite engine evidence, pack/formula semantics, Gas City VCS parity,
backend-aware status, maintenance semantics, and final integration reporting.

The source handoff references two completed gap-analysis workflows:

- `gc-fc8y`: Beads-DoltLite backend audit, summarized in
  `plans/doltlite-gap-fix-build/beads-doltlite-gap-summary.md`.
- `gc-qwr0`: Gas City DoltLite VCS mirroring audit, recorded in
  `plans/doltlite-vcs-mirroring-audit/gap-analysis-report.md`.

The absolute Beads-side source report path named by the handoff was not present
in this checkout at decomposition time. This plan therefore treats the checked
in summary as the canonical Beads-side input and records that absence as an
evidence gap for the final report item.

| Gap | Status |
| --- | --- |
| GAP-001 raw `bd sql` DoltLite writes bypass safety | covered |
| GAP-002 `is_blocked` schema path can go stale | covered |
| GAP-003 linked engine evidence and missing SQL-function diagnostics | covered |
| GAP-004 multiprocess live-city contention coverage | covered |
| GAP-005 `beads-doltlite` pack boundary and role imports | covered |
| GAP-006 Gas City DoltLite VCS workflow parity | covered |
| GAP-007 backend-aware store health, CLI status, and API status | covered |
| GAP-008 DoltLite flatten/gc command and formula failure semantics | covered |
| GAP-009 direct DoltLite VCS parity tests | covered |

## Selected Downstream Formulas

- Build formula: `jj-build`
- Drain policy: `separate`
- Implementation drain formula: `jj-do-work`
- Item implementation formula: `jj-do-work-item`
- Code review formula: `jj-review`
- Review-fix formula: `jj-fix-loop`

Implementation workers must keep workflow documents in the default@ document
workspace and keep source changes in the source workspace selected by the
jj-build setup metadata. Parallel workers must write item-scoped summaries under
`plans/doltlite-gap-fix-build/implementation-items/` and must not use bead
metadata as a document body.

## Implementation Convoy

Implementation convoy: `gc-mgi8`

Convoy name: `doltlite-gap-fix-build-implementation`

The convoy should contain seven runnable work-item beads:

| Key | Title | Trace | Dependencies |
| --- | --- | --- | --- |
| `storage-safety` | Route Beads-DoltLite writes through safe mutation paths | `GAP-001`, `GAP-002`, `GAP-004` | none |
| `linked-engine-evidence` | Prove linked DoltLite engine behavior and diagnostics | `GAP-003`, `GAP-004` | none |
| `pack-maintenance` | Clarify beads-doltlite pack and maintenance semantics | `GAP-005`, `GAP-008` | none |
| `gascity-status` | Make Gas City status and health backend-aware | `GAP-007` | none |
| `vcs-parity` | Add DoltLite VCS workflow parity coverage | `GAP-006`, `GAP-009` | `linked-engine-evidence` |
| `contention-integration` | Add live-city multiprocess contention integration coverage | `GAP-001`, `GAP-003`, `GAP-004`, `GAP-006`, `GAP-007` | `storage-safety`, `linked-engine-evidence`, `gascity-status` |
| `final-report` | Write DoltLite gap-fix implementation report | all gaps | all previous items |

The first four items can run in parallel. `vcs-parity` should wait for linked
engine diagnostics so missing DoltLite SQL functions are reported consistently.
`contention-integration` should wait for the write-safety, engine, and status
surfaces so it tests the final behavior instead of documenting known failure
states. The final report fans in all item summaries.

## Work Items

### `storage-safety`: Route Beads-DoltLite writes through safe mutation paths

Scope:
Fix or explicitly reject Beads-side write paths that bypass DoltLite provider
safety. Start with the `cmd/bd/sql.go` raw SQL path named by the summary, then
audit dependency/status mutation paths that could rely on or update
`is_blocked`.

Expected files and surfaces:

- `cmd/bd/sql.go` in the Beads/DoltLite codebase when available.
- DoltLite accessor and mutation helpers, especially paths that own
  `withDBWrite`, external write locks, stale connection retry/reset behavior,
  typed mutation hooks, and `CommitPending`.
- Migrations or schemas that define `is_blocked`.
- Dependency, status, wisp, parent-child, close, reopen, defer, and delete
  mutation code paths.
- Existing DoltLite tests for raw SQL, blocked status, and dependency queries.

Acceptance criteria:

- Raw `bd sql` writes against DoltLite either route through the provider-owned
  write path or are rejected with a clear error that explains the unsupported
  operation.
- The `is_blocked` field is either removed/deferred from read paths or updated
  by every dependency/status mutation that can change blocked state.
- Tests cover stale blocked-state prevention across at least dependency add,
  dependency close, issue close/reopen, and delete/defer paths.
- Write the item summary to
  `plans/doltlite-gap-fix-build/implementation-items/storage-safety-summary.md`.

Verification expectations:

- Run only focused Beads/DoltLite tests that exercise the changed storage paths.
- If the external Beads source checkout is unavailable, record the exact missing
  path and stop with a clear item failure rather than inventing a local fix.

### `linked-engine-evidence`: Prove linked DoltLite engine behavior and diagnostics

Scope:
Make linked DoltLite test evidence prove the actual linked engine path. The
current handoff notes that focused tests failed to prove engine behavior because
runtime SQL functions such as `dolt_checkout` were unavailable.

Expected files and surfaces:

- `examples/beads-doltlite/commands/build/run.sh`
- `examples/beads-doltlite/health_command_test.go`
- `examples/beads-doltlite/template-fragments/city-understanding.template.md`
- `tools/doltlite-client/`
- DoltLite health and diagnostic command output.
- Focused tests that call or verify `dolt_branch`, `dolt_checkout`,
  `dolt_merge`, `dolt_log`, `dolt_diff_*`, `dolt_hashof_*`, and `dolt_gc()`
  where supported by the linked engine.

Acceptance criteria:

- The build or health diagnostic makes it clear whether the linked binary is
  using libdoltlite and whether required DoltLite SQL functions are registered.
- Missing SQL functions fail with a stable diagnostic that names the unavailable
  function and the linked engine context.
- Focused tests prove the linked engine path instead of only proving fallback or
  process bootstrap behavior.
- Write the item summary to
  `plans/doltlite-gap-fix-build/implementation-items/linked-engine-evidence-summary.md`.

Verification expectations:

- Run the smallest focused tests under `examples/beads-doltlite` and
  `tools/doltlite-client` that prove the linked path.
- Do not run the full Go suite.

### `pack-maintenance`: Clarify beads-doltlite pack and maintenance semantics

Scope:
Make the reusable `beads-doltlite` pack boundary explicit and make maintenance
command failure behavior testable. The Gas City audit found that the pack is not
self-contained for workflow roles and that flatten/gc command failures can look
like successful formula steps.

Expected files and surfaces:

- `examples/beads-doltlite/pack.toml`
- `examples/beads-doltlite/formulas/mol-doltlite-maintenance.toml`
- `examples/beads-doltlite/commands/flatten/run.sh`
- `examples/beads-doltlite/commands/gc/run.sh`
- Any imported role or formula fragments used by the local `city.toml` overlay.

Acceptance criteria:

- The pack is explicitly documented and configured as either provider-only or
  workflow-capable.
- If workflow-capable, it imports or defines the roles/formulas it references;
  if provider-only, maintenance formulas avoid role names that are unavailable
  in the reusable pack.
- Flatten/gc command output has a stable JSON schema for success and failure.
- Formula/order behavior is explicit: non-fatal maintenance records degraded
  state, while fatal maintenance preserves nonzero exit status.
- Tests cover flatten/gc JSON schema, failure behavior, and formula/order event
  metadata.
- Write the item summary to
  `plans/doltlite-gap-fix-build/implementation-items/pack-maintenance-summary.md`.

Verification expectations:

- Run focused command/formula tests for `examples/beads-doltlite`.
- Do not assume the local city overlay imports are present in a reusable pack.

### `gascity-status`: Make Gas City status and health backend-aware

Scope:
Remove Dolt-only assumptions from Gas City store health, CLI status, and API
status when `[beads] backend = "doltlite"` is active.

Expected files and surfaces:

- `cmd/gc/providers.go`
- `cmd/gc/beads_provider_lifecycle.go`
- CLI status and doctor commands that report bead store health.
- API state/status handlers that expose bead backend health.
- Dashboard/generated API surfaces if the API schema changes.
- DoltLite database paths under `.beads/doltlite/*.db`.

Acceptance criteria:

- Dolt-backed stores keep existing `.beads/dolt` health metrics.
- DoltLite-backed stores either omit Dolt-specific health or report
  DoltLite-specific database metrics under a backend-neutral shape.
- CLI and API tests cover `[beads] backend = "doltlite"` so future changes do
  not reintroduce `.beads/dolt` assumptions.
- Any API schema change updates generated OpenAPI/dashboard artifacts through
  the existing project gates.
- Write the item summary to
  `plans/doltlite-gap-fix-build/implementation-items/gascity-status-summary.md`.

Verification expectations:

- Run focused `cmd/gc` tests for status/provider behavior.
- Run `make dashboard-check` only if API schema or dashboard generated files
  change.

### `vcs-parity`: Add DoltLite VCS workflow parity coverage

Scope:
Add direct tests that prove DoltLite-backed Gas City workflow behavior mirrors
the Dolt-backed branch/refinery contract where appropriate.

Expected files and surfaces:

- `cmd/gc` sling, workflow, branch, publish, and refinery tests.
- DoltLite store setup helpers for temporary cities/databases.
- Metadata keys for branch target, source change, branch-ready handoff,
  rejection, and requeue behavior.
- DoltLite SQL functions or diagnostics proven by `linked-engine-evidence`.

Acceptance criteria:

- Focused integration tests cover branch creation and active branch validation.
- Tests cover source-change and target metadata handoff.
- Tests cover non-empty diff/hash detection and empty-diff refusal.
- Tests cover branch-ready metadata and PR/merge empty-branch refusal when the
  Dolt-backed path has matching behavior.
- Tests cover rejection metadata and requeue behavior for failed branch/refinery
  steps.
- Write the item summary to
  `plans/doltlite-gap-fix-build/implementation-items/vcs-parity-summary.md`.

Verification expectations:

- Use temporary DoltLite city/database fixtures.
- If a Dolt-backed behavior should not apply to DoltLite, record the reason and
  add an assertion for the intended DoltLite-specific behavior.

### `contention-integration`: Add live-city multiprocess contention integration coverage

Scope:
Add or update integration coverage that exercises create, update, close, claim,
and ready-work reads against one DoltLite database from multiple processes.

Expected files and surfaces:

- DoltLite-backed Beads provider fixtures.
- `bd` or `doltlite-client` process invocation helpers.
- `gc hook --claim --json` and ready-work read paths.
- Graph/session reconciliation paths that observe assigned and ready work.

Acceptance criteria:

- The test uses more than one process against the same DoltLite database.
- The test covers create, update, close, claim, and ready-work reads.
- The test demonstrates that write safety and stale-handle recovery survive
  concurrent traffic.
- The test avoids status files and discovers running state from live process or
  command results.
- Write the item summary to
  `plans/doltlite-gap-fix-build/implementation-items/contention-integration-summary.md`.

Verification expectations:

- Use an integration build tag if real processes or filesystem-level locks are
  required.
- Keep local verification focused; do not run the full suite.

### `final-report`: Write DoltLite gap-fix implementation report

Scope:
Write the final integration report after all implementation items finish. The
report must reconcile source changes, test evidence, unresolved gaps, and
manifest entries for this build.

Expected files and surfaces:

- All item summaries under
  `plans/doltlite-gap-fix-build/implementation-items/`.
- The updated `plans/doltlite-gap-fix-build/manifest.json`.
- The original handoff and gap summaries.
- Focused test output captured by implementation workers.

Acceptance criteria:

- Write the final report to
  `plans/doltlite-gap-fix-build/final-report.md`.
- Include a gap closure matrix for GAP-001 through GAP-009 with source changes,
  tests run, and residual risk.
- Record any unavailable external Beads report or source checkout as a trace
  limitation, not as hidden evidence.
- Confirm every manifest document entry has path, schema, SHA-256 hash, and jj
  change ID.
- Write the item summary to
  `plans/doltlite-gap-fix-build/implementation-items/final-report-summary.md`.

Verification expectations:

- Validate hashes and paths against the manifest.
- Do not infer workflow completion from source changes alone; use item
  summaries and bead outcomes as the evidence source.
