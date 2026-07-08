# Gas City Workflow Failure Evidence

Date: 2026-07-01
Workspace: `/data/projects/doltlite-gascity/gascity`
Revision: `docs: record Gas City workflow failure evidence`
Tracking bead: `gc-4y91`

## Known Broken Or Questionable Behaviors

1. `jj-build` workflow `gc-wisp-5ek` has not completed. It remains open or
   in progress, with downstream work blocked.

2. A run-operator reported no routed work. We observed work pinned to a
   concrete run-operator session, while ready queries for that operator
   returned no actionable work because dependencies were still open.

3. The control-dispatcher path appears to be the blocker for some workflow
   progress. Several graph.v2 steps route through `core.control-dispatcher`
   before run-operator continuation work can become ready.

4. Existing graph.v2 workflow beads may have stale or bad work-item shape.
   Concrete session assignment on future continuation steps is normal when
   `gc.session_affinity=require`, but here those assigned beads remained
   dependency-blocked and did not surface as ready work.

5. `bd list --include-ephemeral` did not expose expected workflow rows during
   triage. Individual `bd show <id>` calls worked for known workflow beads, but
   broad list/filter queries returned zero in places where rows were expected.
   This may be Beads query behavior, ephemeral scoping, or a storage/index
   issue.

6. Initial `gc sling` attempts routed city-store beads to a rig operator and
   were rejected as cross-store routing. The workflow had to be represented by
   rig-local `gc-*` beads before it could be slung to `gascity/gc.run-operator`.

7. The expected release formula is not present in the formula catalog under the
   name `release`. Related helpers such as `jj-publish` are resolvable, but the
   catalog does not expose the expected release entry.

8. `origin/main` is behind the local `main` bookmark. The remote named `origin`
   does not currently contain the latest local main work.

9. Order-system recovery work created from the Mayor triage remains open,
   including the rig-local workflow root and children.

10. Earlier runtime issues were observed: malformed `gascity-dashboard`
    DoltLite DB, missing `issue_prefix` config errors, failing or stale
    maintenance/order tracking, lingering `orphan-sweep`, and high CPU from
    supervisor/Beads/DoltLite polling plus `send-metrics`.

11. The checkout contains provider API/dashboard diffs unrelated to DoltLite
    backend work or pack/prestart workspace support. Those should be moved
    aside or justified separately.

## Current Investigation Target

Investigate why `bd show <workflow-id>` can read specific workflow beads while
`bd list --include-ephemeral` and metadata-filtered list/ready commands fail to
surface the same rows.

## `bd list --include-ephemeral` Finding

Reproduced on 2026-07-01:

- `gc --rig gascity bd show gc-wisp-896 --json` returns the workflow step.
- `gc --rig gascity bd list --include-ephemeral --id gc-wisp-896 --json`
  returns `[]`.
- `gc --rig gascity bd list --include-ephemeral --metadata-field
  gc.root_bead_id=gc-wisp-5ek --json` returns `[]`.
- `gc --rig gascity bd ready --include-ephemeral --metadata-field
  gc.root_bead_id=gc-wisp-5ek --json` returns seven no-history workflow rows.
- `gc --rig gascity bd list --ready --include-ephemeral --metadata-field
  gc.root_bead_id=gc-wisp-5ek --json` also returns those seven rows.

Source-level cause in `beads-doltlite`:

- `cmd/bd/list_filter.go` maps `--include-ephemeral` to
  `IssueFilter.Ephemeral = true` and `SkipWisps = false`.
- `internal/types/types.go` documents `IssueFilter.Ephemeral` as a selector:
  `nil = any`, `true = only ephemeral`, `false = only persistent`.
- Graph workflow rows such as `gc-wisp-896` are `no_history=true`, not
  `ephemeral=true`. `Issue.Validate` also treats `ephemeral` and `no_history`
  as mutually exclusive.
- The ready-work path uses `WorkFilter.IncludeEphemeral`, whose documented
  behavior is different: ready work includes no-history wisps by default and
  `IncludeEphemeral` adds true ephemeral wisps.

Conclusion: plain `bd list --include-ephemeral` does not mean "include wisps in
addition to normal issues" on the normal list/search path. It currently means
"filter to true ephemeral rows", which excludes graph workflow no-history rows.
For workflow inspection, use `bd ready --include-ephemeral` or
`bd list --ready --include-ephemeral` until the list semantics are fixed.
