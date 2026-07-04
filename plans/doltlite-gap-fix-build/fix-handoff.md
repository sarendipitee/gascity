# DoltLite Gap Fix Build Handoff

## Source Reports

- `gc-fc8y`: beads-doltlite backend gap-analysis, completed with `gc.outcome=pass`.
- `gc-qwr0`: Gas City DoltLite VCS mirroring gap-analysis, completed with `gc.outcome=pass`.
- Beads report: `/data/projects/doltlite-gascity/beads-doltlite/plans/doltlite-backend-upstream-audit/gap-analysis-report.md`.
- Gas City report: `/data/projects/doltlite-gascity/gascity/plans/doltlite-vcs-mirroring-audit/gap-analysis-report.md`.

## Build Request

Use the completed gap reports as the input for a formula-driven fix workflow.
The next workflow should produce requirements, an implementation plan,
decomposition, implementation work, review, and final report for closing the
DoltLite integration gaps.

## Priority Fix Boundaries

1. Raw `bd sql` writes for DoltLite must not bypass DoltLite write safety,
   retry/reset behavior, external locking, or commit semantics.
2. The `is_blocked` DoltLite schema path must be either removed/deferred or
   maintained by dependency/status mutations with parity tests.
3. Linked DoltLite test evidence must prove the actual linked engine path and
   clearly diagnose missing DoltLite SQL functions.
4. Multiprocess/live-city contention coverage must exercise create, update,
   close, claim, and ready-work reads against one DoltLite database.
5. The `beads-doltlite` pack boundary must be made explicit: provider-only or
   workflow-capable, with matching role imports or rewritten maintenance
   formula language.
6. Gas City DoltLite workflow/VCS behavior must mirror the Dolt-backed branch
   and refinery contract where appropriate: branch metadata, target metadata,
   branch-ready handoff, current-branch validation, empty-diff refusal,
   rejection metadata, and requeue behavior.
7. Store health, CLI status, and API status must be backend-aware for
   `.beads/doltlite/*.db` instead of reporting Dolt-only `.beads/dolt` state.
8. DoltLite flatten/gc command and formula failure semantics must be specified
   and tested.
9. Direct DoltLite VCS parity tests must prove branch creation, active branch
   validation, diff/hash non-empty-change detection, branch-ready metadata
   handoff, empty publish/merge refusal, and source workflow metadata
   preservation.

## Constraints

- Do not build, install, or replace live `gc`, `bd`, or `doltlite-client`
  binaries without explicit user approval.
- Do not restart the live city or supervisor unless the workflow explicitly
  records a blocked state requiring operator approval.
- Preserve DoltLite's serverless contract: no Dolt SQL server, runtime port, or
  `.gc/runtime/packs/dolt/dolt-state.json` dependency for DoltLite cities.
- Keep role behavior in pack/formula/prompt assets where possible. Do not
  hardcode DoltLite workflow policy into Go unless the implementation plan
  proves that the boundary belongs there.
- Prefer the `jj-build` formula because this city should keep live beads in
  DoltLite while storing build documents under the `default@` artifact root.
- Leave `push=false` and `open_pr=false` unless the user explicitly authorizes
  publication.

## Expected Output

The formula run should create durable build artifacts under this artifact root,
including requirements, implementation plan, decomposition, implementation
summary, review report, and final report. The decomposition should create
runnable implementation work grouped by the fix boundaries above.
