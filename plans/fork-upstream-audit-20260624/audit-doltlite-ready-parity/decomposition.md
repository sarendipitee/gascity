---
schema: gc.build.decomposition.v1
workflow:
  id: gc-09rm
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
    - path: plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity/requirements.md
      hash: sha256:4150a0e77c0916cdde69931b80339c77da88c11076a2ccdda05ecf6ca78c5e97
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
        - REQ-005
        - REQ-006
    - path: plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity/implementation-plan.md
      hash: sha256:2803afa47f0c4d4f92d123fd70f58940691badd759fbb59237767babdefedc84
  coverage:
    - id: REQ-001
      status: covered
    - id: REQ-002
      status: covered
    - id: REQ-003
      status: covered
    - id: REQ-004
      status: covered
    - id: REQ-005
      status: covered
    - id: REQ-006
      status: covered
---

# Decomposition: Audit DoltLite Ready Parity

## Summary

This decomposition turns the approved DoltLite ready-parity audit plan into one
implementation convoy with three parallel audit lanes and one final fan-in
report item. The convoy performs audit and document work only: it gathers
current evidence, records readiness status, and writes a final
`readiness-audit.md` artifact under the same default@ artifact root.

The plan-review bead was closed before this step, but no `plan-review.md`
document was present in `manifest.json` or under the artifact root at
decomposition time. This decomposition therefore traces to the approved
requirements and implementation plan, and records any review-document absence
as an input gap rather than blocking the audit.

| ID | Status |
| --- | --- |
| REQ-001 | covered |
| REQ-002 | covered |
| REQ-003 | covered |
| REQ-004 | covered |
| REQ-005 | covered |
| REQ-006 | covered |

## Selected Downstream Formulas

- Build formula: `jj-build`
- Drain policy: `separate`
- Implementation drain formula: `jj-do-work`
- Same-session item fallback formula: `jj-do-work-item`
- Code review formula: `jj-review`
- Review-fix formula: `jj-fix-loop`

Each implementation item must keep workflow documents in the default@ document
workspace and use item-scoped implementation-summary paths. Parallel workers
must not write the same summary path or treat bead metadata as a document body.

## Implementation Convoy

Implementation convoy: `gc-tks2`

Convoy name: `audit-doltlite-ready-parity-implementation`

The convoy contains four runnable work-item beads:

| Bead | Title | Trace | Dependencies |
| --- | --- | --- | --- |
| `gc-1ljf` | Audit DoltLite readiness evidence inventory | `REQ-001`, `REQ-002`, `REQ-006` | none |
| `gc-1m8v` | Map Dolt regression coverage | `REQ-001`, `REQ-004`, `REQ-005` | none |
| `gc-vkeh` | Audit DoltLite provider boundaries and operations | `REQ-003`, `REQ-004`, `REQ-005` | none |
| `gc-m6j2` | Write DoltLite readiness audit report | `REQ-001`, `REQ-002`, `REQ-003`, `REQ-004`, `REQ-005`, `REQ-006` | `gc-1ljf`, `gc-1m8v`, `gc-vkeh` |

The first three items may run in parallel. The final report item is ordered
behind them so it can integrate their evidence into the canonical
`readiness-audit.md` artifact and manifest entry.

## Work Items

### `gc-1ljf`: Audit DoltLite readiness evidence inventory

Scope:
Build the current evidence inventory for the audit. Start from
`engdocs/contributors/dolt-regression-audit.md` and the requirement checklist,
then confirm every cited path against the current checkout before treating it
as evidence. Include current code, tests, docs, scripts, schemas, and commands
relevant to REQ-001 through REQ-006.

Expected files and surfaces:

- `engdocs/contributors/dolt-regression-audit.md`
- `engdocs/archive/analysis/feature-parity.md`
- `engdocs/archive/analysis/gastown-upstream-audit.md`
- `cmd/gc/providers.go`
- `cmd/gc/beads_provider_lifecycle.go`
- `cmd/gc/bd_env.go`
- `cmd/gc/dolt_runtime_publication.go`
- `internal/beads/`
- `internal/beads/contract/`
- `internal/beads/exec/`
- `examples/beads-doltlite/`
- `tools/doltlite-client/`
- `schemas/beads-doltlite/health/result.schema.json`

Acceptance criteria:

- Write the item summary to
  `plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity/implementation-items/evidence-inventory-summary.md`.
- List exact files, tests, scripts, schemas, and commands that later audit
  lanes should inspect.
- Mark missing, stale, or uncertain paths as gaps instead of inferring them from
  historical documents.
- Make no source changes.

Verification expectations:

- Use targeted file/path checks and focused command discovery.
- Do not run the full Go test suite.

### `gc-1m8v`: Map Dolt regression coverage

Scope:
Build the regression coverage matrix required by REQ-001. Cover every known
Dolt regression class named by the approved plan and assign `covered`,
`partial`, `missing`, or `not applicable` with current evidence.

Regression classes:

- `GC_DOLT_PORT` versus `BEADS_DOLT_PORT` drift.
- Stale runtime state and stale port-file rejection.
- Stale ambient `BEADS_*` and Dolt environment stripping.
- Duplicate lifecycle actions and Dolt restart races.
- Unusable `.beads` bootstrap or stale `issues.jsonl` state.
- Orphaned Dolt SQL servers serving deleted or stale data.
- Missing `exec:gc-beads-bd` CRUD and ready behavior.
- Managed session `GC_BEADS=exec:gc-beads-bd` routing.
- DoltLite native read/write fast path behavior and fallback boundaries.

Acceptance criteria:

- Write the item summary to
  `plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity/implementation-items/regression-coverage-summary.md`.
- Include a matrix with status, evidence paths, focused checks run or not run,
  and follow-up gaps.
- Mark expensive or unavailable live-Dolt checks as `not run` with a reason and
  cite static evidence where possible.
- Make no source changes.

Verification expectations:

- Use the approved focused commands for `cmd/gc`, `internal/beads`,
  `internal/beads/contract`, `internal/beads/exec`, and
  `examples/beads-doltlite` when feasible.
- Do not run the full Go test suite.

### `gc-vkeh`: Audit DoltLite provider boundaries and operations

Scope:
Review provider-boundary isolation and operational readiness. Classify
DoltLite, T3, and fork-specific behavior by owner boundary: provider, runtime,
config, pack, docs, or generic SDK. Treat generic-SDK leakage as a finding when
DoltLite or T3 assumptions appear outside the approved boundaries.

Expected files and surfaces:

- `cmd/gc/providers.go`
- `cmd/gc/store_target_exec_test.go`
- `cmd/gc/hook_cross_store.go`
- `cmd/gc/api_state_test.go`
- `cmd/gc/beads_provider_lifecycle.go`
- `cmd/gc/bd_env.go`
- `cmd/gc/dolt_runtime_publication.go`
- `cmd/gc/dolt_start_managed_test.go`
- `cmd/gc/dolt_lifecycle_race_test.go`
- `cmd/gc/cmd_stop_test.go`
- `internal/beads/factory.go`
- `internal/beads/exec/`
- API/dashboard bead read paths
- `examples/beads-doltlite/`

Acceptance criteria:

- Write the item summary to
  `plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity/implementation-items/provider-operations-summary.md`.
- Name each boundary finding with owner boundary, evidence, severity, and
  recommended next action.
- Explicitly call out operational gaps that could make `bd`, `gc hook`,
  graph.v2 dispatch, or API bead reads unreliable.
- Make no source changes.

Verification expectations:

- Use focused checks from the approved plan only when they directly support a
  boundary or operational claim.
- Record unavailable live-Dolt prerequisites instead of inferring readiness.
- Do not run the full Go test suite.

### `gc-m6j2`: Write DoltLite readiness audit report

Scope:
Integrate the lane summaries into
`plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity/readiness-audit.md`.
This is the canonical final audit artifact for the workflow root.

Report sections:

- Regression coverage.
- Provider-boundary findings.
- Operational readiness.
- Verification commands, including skipped or unavailable checks.
- Follow-up gaps grouped as source fixes, tests, documentation, or no-op
  confirmations.

Acceptance criteria:

- Write the final audit report under the default@ artifact root.
- Update `manifest.json` with a `readiness-audit` document entry using schema
  `markdown`, SHA-256 content hash, and jj document change ID.
- Write the item summary to
  `plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity/implementation-items/readiness-audit-summary.md`.
- Cite concrete current evidence for every readiness claim and record
  uncertainty as a gap.
- Do not create follow-up source-fix beads; identify candidates only.
- Make no source changes.

Verification expectations:

- Verify hashes and manifest entries after writing the report.
- Do not run the full Go test suite.

## Skipped And Blocked Work

- Source fixes, new tests, dashboard/API changes, and PR work are out of scope
  for this audit convoy unless a later approved task scopes them in.
- Follow-up implementation beads are intentionally deferred until the final
  audit proves a current gap with evidence.
- The missing plan-review document is recorded as an input gap. It does not
  block the decomposition because the requirements and implementation plan are
  approved and manifest-backed.
