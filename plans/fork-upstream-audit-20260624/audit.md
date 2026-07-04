---
plan_slug: fork-upstream-audit-20260624
phase: fork-audit
rig: gascity
rig_root: /data/projects/doltlite-gascity/gascity
artifact_root: /data/projects/doltlite-gascity/gascity/plans
status: draft
created_at: 2026-06-24T16:34:00Z
updated_at: 2026-06-24T16:34:00Z
---

# Fork Upstream Audit: Gas City

## Goal

Systematically review all areas changed on the local fork against upstream
`gastownhall/gascity`, preserve every local change, and document findings before
any rebase, restore, cleanup, or upstream reconciliation.

## Preservation Rules

- Do not run `jj rebase`, `jj restore`, `jj abandon`, `git checkout`, `git reset`,
  or any destructive cleanup while auditing.
- Use `jj` for read-only repository inspection. This repo is colocated
  `.jj` + `.git`; raw Git mutations can desynchronize the working copy.
- Treat runtime files and Beads state as data to classify, not noise to delete.
- Before any future reconciliation work, create explicit preservation refs or
  exported patches for the current line. This audit has not created those refs.

## Baseline

- Upstream remote: `upstream https://github.com/gastownhall/gascity`
- Fork remote: `origin git@github.com:duncan4123/gascity.git`
- Upstream `main`: `pmkksnuywmkw e22049f86666` -
  `Fail closed pool creates on partial scale_check reads (#3686)`
- Current workspace `@`: `vvytppnruwyl 06fbcbd23133` -
  `merge: combine xvsqtzym and kkzqlryk lines`
- Running `gc` binary observed earlier: `614d98bf78e6-dirty`, resolving locally to
  `kkzqlryksowp` - `document: write jj review report`
- Current working copy already had runtime/local churn before this audit:
  `.beads/config.yaml`

## Diff Inventory

Comparison base: `jj diff --git --from main --to @`.

| Area | Files | Added | Deleted | Initial Risk |
| --- | ---: | ---: | ---: | --- |
| `gc controller/cli` | 63 | 3768 | 762 | High |
| `beads/doltlite store` | 21 | 2945 | 1065 | High |
| `docs/plans/reviews` | 47 | 2601 | 404 | Medium |
| `beads-doltlite pack` | 32 | 2218 | 0 | High |
| `tools/doltlite-client` | 1 | 464 | 0 | Medium |
| `api/event export` | 15 | 104 | 245 | Medium |
| `tests` | 5 | 195 | 53 | Medium |
| `config/imports` | 10 | 135 | 20 | High |
| `runtime/workdir` | 6 | 120 | 11 | High |
| `doctor/diagnostics` | 10 | included above | included above | Medium |

Largest changed files:

- `internal/beads/doltlite_read_store.go`: +1170 / -412
- `internal/beads/doltlite_read_store_test.go`: +1093 / -444
- `examples/beads-doltlite/commands/build/run.sh`: +1016 / -0
- `docs/reference/exec-beads-provider.md`: +550 / -176
- `cmd/gc/build_desired_state_test.go`: +354 / -188
- `tools/doltlite-client/main.go`: +464 / -0
- `cmd/gc/build_desired_state.go`: +365 / -31
- `cmd/gc/build_stamp.go`: +222 / -0
- `cmd/gc/order_dispatch_test.go`: +195 / -0
- `internal/beads/bdstore.go`: +150 / -11

## Initial Findings

### F-001: Running Binary Does Not Match Upstream or Clean Current Source

The controller was running a dirty binary at `614d98bf78e6-dirty`. Upstream
`main` is `e22049f86666`; current source `@` is `06fbcbd23133`.

Impact: observed runtime behavior may come from a dirty build that is neither
clean upstream nor clean current workspace. Any bug fix must first pin which
source revision produced the binary being tested.

Recommended next step: create a build provenance note and rebuild a clean `gc`
from the chosen preserved revision before validating fixes.

### F-002: Controller Demand Path Diverges From Upstream

Touched files include:

- `cmd/gc/build_desired_state.go`
- `cmd/gc/build_desired_state_test.go`
- `cmd/gc/city_runtime.go`
- `cmd/gc/pool_desired_state_test.go`

Current symptom: direct `bd ready` and `gc hook` see worker-routed work, but
controller trace `scale_check_counts` only contains:

- `beads-doltlite/core.control-dispatcher`
- `gascity/core.control-dispatcher`

Impact: demand-driven workers remain idle even when routed work exists.

Working hypothesis: local DoltLite/controller changes diverge from upstream
pool-demand semantics. The audit must compare `defaultScaleCheckCountsAndDemand`,
`readyForControllerDemand`, demand snapshots, and pool desired-state merging
against upstream `main`, then add a regression test using normal worker routes,
not only control-dispatcher routes.

### F-003: DoltLite Native Store Path Is the Largest and Highest-Risk Divergence

Touched files include:

- `internal/beads/doltlite_read_store.go`
- `internal/beads/doltlite_read_store_test.go`
- `internal/beads/bdstore.go`
- `internal/beads/doltlite_count.go`
- `internal/beads/factory.go`
- `internal/beads/beads.go`

Impact: the native read path can diverge from CLI `bd ready`. That is exactly
the class of failure suspected in F-002. The build tag
`gascity_doltlite_lib` also means ordinary untagged tests can miss the active
runtime path.

Recommended next step: add parity tests comparing native `Ready(TierBoth)` and
CLI `bd ready --include-ephemeral` for routed, unassigned worker tasks with
non-blocking `tracks` dependencies.

### F-004: Pack-Routed Pool Workdirs and Trigger Metadata Are Entangled With Runtime Failures

Touched areas include pool trigger metadata, configured workdirs, pack workspace
reuse, and workdir protection.

Observed runtime failures:

- `gascity-packs/packer.packsmith` repeatedly failed on stale JJ workspace
  registration for workspace `packer`.
- `core.control-dispatcher` quarantined beads because check paths resolved
  under per-bead work dirs such as
  `.../gc-rg3l-generate-requirements/.gc/scripts`, which do not exist.

Impact: even when the controller starts sessions, workdir/path resolution can
quarantine workflow beads or prevent workers from starting.

Recommended next step: review local workdir and trigger-bead changes against
upstream path-resolution invariants, then add tests for formula check paths and
pack workspace reuse after stale metadata is present.

### F-005: Config/Import Changes Do Not Migrate Existing Beads

Touched files include:

- `internal/config/config.go`
- `internal/config/compose.go`
- `internal/config/pack.go`
- `internal/config/patch.go`
- related config tests

The switch to `gascity-jj-base` is installed and `gc import check` passes, but
existing beads routed to older targets such as `gascity/gc.*` and
`gc.run-operator` are not rewritten by import installation or reload.

Impact: stale routes remain unworkable until a migration map is applied or the
affected workflow roots are regenerated.

Recommended next step: build a route migration report before changing any
beads. The report should map stale targets to either a replacement target,
manual closure, or regeneration.

### F-006: Fork Diff Contains Generated Docs, Review Artifacts, Runtime State, and Scratch Programs

Examples:

- `gc-plans/github/pulls/.../review-report.md`
- generated schema/reference docs
- `.beads/config.yaml`
- `cmd/tmpdebug/main.go`
- `tmpinspect/main.go`

Impact: these make upstream comparison noisy and can hide real source
divergence. They must be classified before any upstream merge work.

Recommended next step: split the audit inventory into source, tests, docs,
generated artifacts, runtime state, and scratch/debug files. Do not delete or
restore anything until ownership is known.

### F-007: Runtime Demand Snapshot Work Has Divergent Local/Origin Variants

Bookmark state includes divergent `pr/runtime-ready-demand-snapshot` revisions:

- local: `bda1fd03` - `fix(runtime): refresh demand snapshots for routed work`
- origin: `4a6be657` - same subject, divergent content

Impact: upstream reconciliation must choose or merge the correct variant before
building further runtime-demand fixes.

Recommended next step: compare the divergent revisions and document which one
matches the currently running behavior.

## Review Queue

1. Controller demand and pool desired state
   - Files: `cmd/gc/build_desired_state.go`, `cmd/gc/city_runtime.go`, related
     tests.
   - Output: root-cause note for `scale_check_counts` missing worker templates.

2. DoltLite native read/write path
   - Files: `internal/beads/doltlite_read_store.go`, `bdstore.go`,
     `doltlite_count.go`, factory wiring.
   - Output: CLI/native parity matrix and missing regression tests.

3. Pack-routed pool and workdir handling
   - Files: pool trigger/workdir code, `internal/workdir`, packsmith scripts.
   - Output: stale workspace and per-bead `.gc/scripts` path findings.

4. Config/import/pack composition
   - Files: `internal/config/*`, pack import patches.
   - Output: map of old route names to current configured agents.

5. Beads-doltlite pack and diagnostics
   - Files: `examples/beads-doltlite/*`, doctor checks, build/client scripts.
   - Output: pack build/runtime health checklist.

6. Generated docs, review artifacts, scratch/debug files
   - Files: docs/reference, gc-plans, `cmd/tmpdebug`, `tmpinspect`, `.beads`.
   - Output: keep/drop/classify list. No deletion during audit.

## Proposed Follow-Up Beads

Do not create these until approved.

1. `audit-controller-demand-vs-upstream`
   - Compare local controller demand code with upstream `main`.
   - Add/identify regression tests for normal worker routed demand.

2. `audit-doltlite-ready-parity`
   - Compare native DoltLite `Ready()` semantics to `bd ready`.
   - Cover routed worker tasks, wisps/no-history rows, metadata parsing, and
     `tracks` dependencies.

3. `audit-pack-workdir-path-resolution`
   - Reproduce packsmith stale workspace and dispatcher check-path quarantine.
   - Identify whether the bug is in local trigger metadata, workdir reuse, or
     formula path resolution.

4. `audit-route-migration-map`
   - Inventory stale `gc.routed_to` and `gc.run_target` values.
   - Produce a no-mutation migration proposal.

5. `audit-fork-artifact-classification`
   - Classify generated docs, runtime metadata, and scratch tools so later
     upstream reconciliation can preserve real work without carrying noise.

