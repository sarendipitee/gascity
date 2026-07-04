# DoltLite VCS Mirroring Gap Analysis Report

Subject: `/data/projects/doltlite-gascity/gascity/plans/doltlite-vcs-mirroring-audit/subject.md`
Subject SHA-256: `627f7fabf61b6f1e7ecb61bbe71da4132e7b47ca49f5f59bccfec43e984883d7`
Audit date: 2026-06-23

## Verdict

FAIL - changes required.

The DoltLite storage/backend integration is substantially present and has good
coverage around serverless operation, doctor/startup behavior, and the
in-process read path. It does not yet mirror the Dolt-backed Gas City workflow
and VCS behavior described in the subject. The main gaps are at the pack/formula
boundary and the VCS handoff/publish boundary, not in basic DoltLite bead
storage.

In particular, importing the generic `gc` roles into the local
`beads-doltlite` rig makes planning/build/review workers available in this city,
but it does not provide the Gastown-style branch/refinery workflow: per-bead
branch metadata ownership, branch-ready handoff, target branch metadata,
false-completion diff checks, PR/merge handling, and rejection recovery.

## Scope And Evidence

Reviewed evidence included:

- `/data/projects/doltlite-gascity/city.toml`
- `/data/projects/doltlite-gascity/pack.toml`
- `/data/projects/doltlite-gascity/packs.lock`
- `gascity-packs/gascity/roles`
- `gascity-packs/gastown/formulas/mol-polecat-work.toml`
- `gascity-packs/gastown/formulas/mol-refinery-patrol.toml`
- `gascity-packs/gastown/agents/polecat/prompt.template.md`
- `gascity-packs/gastown/agents/refinery/prompt.template.md`
- `gascity-packs/gastown/template-fragments/approval-fallacy.template.md`
- `gascity-packs/gascity/assets/workflows/do-work/*`
- `gascity-packs/gascity/assets/workflows/publish/*`
- `examples/beads-doltlite/*`
- `examples/bd/dolt/*`
- `cmd/gc/beads_provider_lifecycle.go`
- `cmd/gc/beads_backend_test.go`
- `cmd/gc/cmd_doctor_extract_test.go`
- `cmd/gc/cmd_doctor_test.go`
- `internal/sling/sling.go`
- `internal/sling/sling_core.go`
- `internal/sling/sling_test.go`
- `internal/storehealth/storehealth.go`
- `internal/api/store_health.go`
- `cmd/gc/store_health.go`
- `internal/beads/doltlite_read_store.go`
- `internal/beads/doltlite_read_store_test.go`
- `/data/projects/doltlite-gascity/doltlite/README.md`

The dependency context bundle for this bead was absent. This report is based on
the subject file and the current checked-out files above.

## Mirrored Correctly

1. Local city overlay imports the `gc` role pack for both `gascity` and
   `beads-doltlite`.

   The local `/data/projects/doltlite-gascity/city.toml` now gives the
   `beads-doltlite` rig the same `gascity/roles` import as the `gascity` rig.
   That corrects the local "no gc roles available" symptom described in the
   subject. This is a city overlay fact, not a property of the embedded
   `examples/beads-doltlite` pack.

2. DoltLite avoids managed Dolt server assumptions.

   `cmd/gc/beads_provider_lifecycle.go` skips managed server startup when the
   resolved backend does not need a managed server. The doctor tests verify
   DoltLite uses pack-managed checks and does not register legacy Dolt checks
   such as `dolt-server`, `dolt-topology`, `dolt-drift`, or `dolt-noms-size`.
   This is the correct non-mirroring of Dolt SQL server/runtime port behavior.

3. DoltLite health is serverless and tested.

   `examples/beads-doltlite/commands/health/run.sh` uses `bd status --json`
   under `BEADS_BACKEND=doltlite` / `GC_BEADS_BACKEND=doltlite`, with optional
   timeout configuration and JSON fallback behavior. Its tests cover the schema,
   lack of a hardcoded default timeout, and operation without `jq`.

4. Default target branch resolution is backend-neutral and tested.

   `internal/sling.SlingFormulaTargetBranch` resolves target branches from work
   bead metadata, then rig defaults, then agent rig defaults, then a live branch
   probe. The tests cover bead metadata, bead-prefix rig defaults, hyphenated
   prefixes, agent rig defaults, and live-probe fallback.

5. Graph v2 source workflow launch has DoltLite lock recovery coverage.

   `internal/sling` includes recovery for transient instantiate errors after a
   partially materialized Graph v2 root. `TestInstantiateSlingFormulaRecoversGraphV2RootAfterDoltLiteLock`
   simulates a `database is locked` failure and expects recovery to one root.

6. DoltLite read-store coverage is strong.

   `internal/beads/doltlite_read_store.go` provides an in-process hot read path
   for DoltLite-backed stores. Tests cover listing, ready projection, stale Dolt
   metadata handling, canonical dependency schema, in-process durable/wisp
   writes, cache behavior, and concurrency-sensitive paths.

## Intentional Non-Parity

These Dolt behaviors should not be copied into DoltLite:

- Managed Dolt SQL server lifecycle, runtime ports, TCP health checks, and
  `.gc/runtime/packs/dolt/dolt-state.json`.
- Dolt remote patrol and backup assumptions that depend on a managed server
  topology.
- Heavyweight flatten or GC operations during startup.
- Startup-time compaction as a health requirement.

DoltLite should expose health and maintenance through its embedded
`.beads/doltlite/*.db` files, pack-managed doctor checks, and explicit
maintenance commands or orders.

## Findings

### F1 - `examples/beads-doltlite` is not self-contained for workflow roles

Severity: high

Evidence:

- `/data/projects/doltlite-gascity/city.toml` imports `gascity/roles` into the
  local `beads-doltlite` rig.
- `examples/beads-doltlite/pack.toml` imports only `bd`; it has no `gc` import
  and no `agents/` directory.
- `examples/beads-doltlite/formulas/mol-doltlite-maintenance.toml` describes
  itself as "run by dog pool".
- The `dog` agent exists under `examples/bd/dolt/agents/dog`, not under
  `examples/beads-doltlite` or base `examples/bd`.

Impact:

The local city overlay can route generic `gc` work today, but the reusable
`beads-doltlite` pack is not independently runnable for workflow work that
expects a maintenance agent. A city that imports `beads-doltlite` without the
same local role overlay may have formulas/orders that refer to an agent pool the
pack does not provide.

Recommended fix:

Choose and document the pack boundary.

- If `beads-doltlite` is a pure provider pack, remove or rewrite dog-pool
  formula language and keep maintenance as exec-only orders.
- If `beads-doltlite` is expected to run formula workers, add or import the
  operational role pack explicitly and add a catalog/routing test that proves
  every DoltLite formula can be assigned in a city that imports only the stated
  packs.

### F2 - VCS/refinery workflow parity is missing from the DoltLite route

Severity: high

Evidence:

- Gastown's Dolt-backed path in `mol-polecat-work.toml`,
  `mol-refinery-patrol.toml`, and the polecat/refinery prompts has a concrete
  VCS contract:
  - deterministic per-bead branch names such as `polecat/<bead-id>`;
  - `metadata.branch` and `metadata.target` ownership;
  - branch creation from the resolved base branch;
  - current-branch and branch-shape validation;
  - `branch_ready=true` handoff when auto-push is disabled;
  - refinery-side rebase, gate execution, PR/direct merge handling, and
    rejection metadata;
  - false-completion refusal based on a real diff check, not commit count alone.
- The imported `gascity` role pack provides generic implementation,
  review, publishing, and planning roles, but its current workflows mostly
  create detached git worktrees and source-anchor metadata. The reviewed
  `do-work` and `publish` assets do not implement the Gastown branch/refinery
  handoff contract.

Impact:

DoltLite work can get generic implementation workers, but it does not get the
Dolt-backed VCS behavior the subject asks to mirror. This is the largest parity
gap: the storage backend can work while the workflow still lacks the branch,
handoff, PR/merge, and false-completion safeguards.

Recommended fix:

Decide whether the branch/refinery contract belongs in the `gascity` generic
methodology, in a separate VCS methodology pack, or in a DoltLite-specific pack
that imports a shared VCS workflow. Then port the smallest proven slice of the
Gastown behavior:

- branch metadata contract (`branch`, `target`, `branch_ready`,
  `halt_reason`);
- branch creation and current-branch validation;
- diff-based non-empty-change guard;
- publish/merge refusal for empty branches;
- rejection metadata and requeue behavior.

Keep this in pack/formula/prompt assets. Do not hardcode role names or VCS
judgment in Go.

### F3 - Generic store health is still Dolt-path specific

Severity: medium-high

Evidence:

- `internal/storehealth/storehealth.go` describes "Dolt bead store health" and
  hardcodes `cityPath/.beads/dolt`.
- `cmd/gc/store_health.go` and `internal/api/store_health.go` call that path
  without an observed backend-specific guard.
- Existing CLI/API tests assert the `.beads/dolt` path suffix.

Impact:

For a DoltLite city, `gc status` and API status can report irrelevant or
misleading Dolt store health instead of DoltLite health from
`.beads/doltlite/*.db`, or they can show a zero-sized missing Dolt path as if it
were the active store.

Recommended fix:

Make store health backend-aware:

- Dolt: keep the existing `.beads/dolt` metrics.
- DoltLite: either omit Dolt store health entirely or report DoltLite-specific
  database file metrics under a backend-neutral shape.
- Add CLI and API tests for `[beads] backend = "doltlite"` so future changes do
  not reintroduce Dolt-only status assumptions.

### F4 - DoltLite maintenance command/formula semantics are under-specified

Severity: medium

Evidence:

- `examples/beads-doltlite/commands/flatten/run.sh` and
  `examples/beads-doltlite/commands/gc/run.sh` convert command failures into
  JSON objects and return shell success because the fallback `echo` succeeds.
- `examples/beads-doltlite/orders/doltlite-maintenance.toml` runs
  `gc beads-doltlite flatten --json && gc beads-doltlite gc --json`.
- `mol-doltlite-maintenance.toml` records maintenance events and closes the
  bead, but no focused tests were found for flatten/gc command failure
  semantics or formula-level error propagation.

Impact:

Maintenance may look successful to order dispatch even when flatten or GC failed
and only emitted an error JSON object. That may be acceptable for non-fatal
maintenance, but it is not explicit enough for operators or tests.

Recommended fix:

Define the command contract and test it:

- If maintenance failures are non-fatal, return `ok=false` with a stable error
  field and make the formula/order record degraded maintenance explicitly.
- If failures should fail the order, preserve the nonzero exit status.
- Add tests for flatten/gc JSON schema, failure behavior, and formula/order
  event metadata.

### F5 - Direct DoltLite VCS parity tests are missing

Severity: medium

Evidence:

- The DoltLite README documents branch-aware SQLite database opening,
  `dolt_branch`, `dolt_checkout`, `dolt_merge`, `dolt_log`, `dolt_diff_*`,
  `dolt_hashof_*`, and `dolt_gc()`.
- The Gas City tests reviewed cover DoltLite storage, health, doctor behavior,
  sling branch resolution, and lock recovery.
- No focused test was found that proves a DoltLite-backed workflow performs
  branch creation/current-branch validation, non-empty diff checks, branch-ready
  metadata handoff, or PR/merge empty-branch refusal.

Impact:

The DoltLite substrate appears capable of the needed VCS operations, but Gas
City does not yet prove that its DoltLite workflow path uses those operations to
match the Dolt-backed behavior.

Recommended fix:

Add focused integration tests against a temporary DoltLite city/database. Cover:

- creating a branch and validating the active branch;
- recording `metadata.branch` and `metadata.target`;
- detecting a real diff with DoltLite SQL functions rather than commit count;
- refusing empty branch publish/merge;
- running `SELECT dolt_gc();` or the supported DoltLite GC command path through
  the maintenance wrapper.

### F6 - Source-change tracking evidence is incomplete

Severity: low-medium

Evidence:

- `internal/sourceworkflow` and `internal/sling` provide backend-neutral source
  workflow tracking and source-launch visibility/recovery behavior.
- The reviewed `gascity` workflows preserve source-anchor metadata and detached
  worktree discipline.
- No direct DoltLite evidence was found for source-change metadata parity with
  the Dolt-backed workflow path described in the subject.

Impact:

Source workflow tracking may be sufficient, but the audit did not find a
conclusive DoltLite-specific proof that source changes are carried through the
same workflow metadata and handoff points as the Dolt-backed route.

Recommended fix:

Add a small DoltLite graph-v2 source workflow test that asserts the source bead,
source workflow metadata, work directory, target branch, and branch metadata
survive the full implementation-to-publish path.

## Missing Evidence

- No dependency context bundle was available.
- No live DoltLite formula run was executed for this report.
- No generated formula catalog or resolved routing table was exhaustively
  inspected.
- No end-to-end proof was found that a `beads-doltlite` rig can run local build
  work independently of the `gascity` rig's local city overlay.
- No evidence was found that `mol-doltlite-maintenance` can route to an actual
  agent in a minimal city that imports `beads-doltlite`.
- No evidence was found that CLI/API store health is backend-gated for
  DoltLite.
- No focused DoltLite VCS workflow tests were found for branch creation,
  current-branch validation, false-completion checks, or publish/merge refusal
  of empty branches.

## Recommended Follow-Up Beads

### Pack And Role Boundary

- Decide whether `examples/beads-doltlite` is provider-only or workflow-capable.
- If provider-only, remove/replace dog-pool maintenance formula references and
  keep maintenance as pack-managed exec orders.
- If workflow-capable, import or define the required operational role pack and
  add a catalog test proving DoltLite formulas have routable agents.

### VCS Workflow Boundary

- Create a backend-neutral VCS methodology pack or port the Gastown
  branch/refinery contract into `gascity` workflow assets.
- Add branch metadata, target metadata, branch-ready, empty-diff refusal, and
  rejection metadata to the chosen workflow path.
- Keep role behavior in configuration and prompt/formula assets, not hardcoded
  Go branches.

### Status And Health Boundary

- Make `internal/storehealth`, CLI status, and API status backend-aware.
- Add tests for DoltLite city status/API responses so `.beads/dolt` does not
  appear as the active store path.

### Maintenance Boundary

- Specify flatten/gc command success and failure contracts.
- Add tests for health, flatten, gc, and maintenance order/formula behavior.
- Ensure maintenance never runs heavyweight flatten/GC during startup.

### DoltLite VCS Tests

- Add focused tests proving DoltLite branch creation, active branch validation,
  diff/hash non-empty-change detection, branch-ready metadata handoff, and
  empty publish/merge refusal.
- Include one source-workflow test that exercises source metadata through the
  DoltLite implementation-to-publish path.

## Bottom Line

DoltLite is integrated well as a serverless bead storage backend. It is not yet
integrated as a full mirror of the Dolt-backed Gas City workflow/VCS path. The
next work should focus on pack boundaries, workflow contracts, and targeted
parity tests rather than on replacing the storage substrate.
