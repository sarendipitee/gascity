---
schema: gc.build.plan.v1
workflow:
  id: gc-9d37
  formula: jj-build
methodology:
  pack: gascity-jj-base
  name: jj-build
producer:
  formula: jj-build
  stage: plan
  attempt: 2
plan_slug: controller-formula-doltlite-query-write-audit-gascity
phase: implementation-plan
rig: gascity
rig_root: /data/projects/doltlite-gascity/gascity
artifact_root: /data/projects/doltlite-gascity/gascity/plans/controller-formula-doltlite-query-write-audit-gascity
status: blocked
created_at: 2026-06-28T12:20:00Z
updated_at: 2026-06-28T12:20:00Z
trace:
  upstream:
    - path: plans/controller-formula-doltlite-query-write-audit-gascity/manifest.json
      status: observed
    - path: plans/controller-formula-doltlite-query-write-audit-gascity/implementation-summary.md
      status: observed
    - path: plans/controller-formula-doltlite-query-write-audit-gascity/final-report.md
      status: observed
    - path: plans/controller-formula-doltlite-query-write-audit-gascity/review.md
      status: observed
  coverage:
    - id: REQ-SOURCE-WORKSPACE
      status: blocked
    - id: REQ-SOURCE-CHANGE
      status: blocked
    - id: REQ-COMPLETE-DOC-HANDOFF
      status: covered
    - id: REQ-DOLTLITE-QUERY-WRITE-AUDIT
      status: deferred
    - id: REQ-NO-SOURCE-CHANGES
      status: covered
---

# Implementation Plan: Controller Formula DoltLite Query/Write Audit

## Summary

Produce a repair-oriented audit plan for workflow `gc-9d37`, a `jj-build`
run intended to evaluate controller formula behavior against the DoltLite
query/write path. The current managed artifact state is incomplete: the
manifest records `source_workspace`, `source_workspace_path`, and
`source_change_id` as `missing`, and the artifact root does not contain the
expected requirements, decomposition, item implementation summaries, or source
workspace evidence.

This plan therefore does not authorize source edits. It defines the smallest
next implementation convoy needed to recover a reviewable audit: restore the
document handoff, identify the real source workspace and source change, collect
evidence for controller formula query/write behavior with DoltLite-backed beads,
and write a manifest-managed audit report. Live bead state stays in DoltLite;
document bodies stay under the `default@` artifact root.

## Current System

The workflow root is `gc-9d37`, using formula `jj-build` with input convoy
`gc-dip0` and document artifacts under
`plans/controller-formula-doltlite-query-write-audit-gascity`.

The current artifact root contains these managed documents:

- `implementation-summary.md`, schema `gc.build.implementation-summary.v1`
- `final-report.md`, schema `gc.build.final-report.v1`
- `review.md`, schema `gc.build.review.v1`
- `manifest.json`, schema `gascity.workflow-docs.v1`

The manifest records the document workspace as `default`, the workspace path as
`/data/projects/doltlite-gascity/gascity`, and the document base revset as
`default@`. It also records `source_workspace`, `source_workspace_path`, and
`source_change_id` as `missing`. That state matches the implementation summary
and review: downstream review cannot verify a controller formula DoltLite
query/write implementation without a resolvable source change.

Relevant source and workflow surfaces for the eventual audit include:

- `.beads/formulas/jj-build.formula.toml`
- `.beads/formulas/jj-planning-base.formula.toml`
- `.beads/formulas/jj-decomposition-base.formula.toml`
- `cmd/gc/controller.go` and controller tests
- `cmd/gc/doltlite_store_native.go` and `cmd/gc/doltlite_store_native_test.go`
- `cmd/gc/bead_policy_store_test.go`
- `internal/beads/doltlite_read_store.go`
- `internal/beads/doltlite_count.go`
- `examples/beads-doltlite/`
- `tools/doltlite-client/`

The artifact set is currently a workflow recovery target, not evidence that the
source implementation exists or passed review.

## Proposed Implementation

1. Preserve the managed document handoff.

   Keep `manifest.json` as the authoritative list of workflow documents. Each
   document-producing step must record `name`, `schema`, `path`,
   `absolute_path`, `hash`, and `change_id`. Bead metadata should contain only
   paths, schema IDs, hashes, and change IDs.

2. Recover source identity before source review.

   Find the actual jj source workspace and source change that contain the
   controller formula DoltLite query/write work. Record them as
   `source_workspace`, `source_workspace_path`, and `source_change_id` in the
   manifest and matching `gc.docs.source_*` metadata. Do not reuse document
   change IDs as source change IDs.

3. Reconstruct missing upstream documents if they cannot be recovered.

   If the original requirements and decomposition artifacts are unavailable,
   write replacement managed documents that explicitly mark the recovered scope.
   The recovered requirements should focus on:

   - Controller formula execution can query DoltLite-backed bead state through
     the normal beads abstractions.
   - Controller formula execution can write routed workflow state through the
     provider-owned write path.
   - The audit distinguishes live DoltLite bead data from jj-managed build
     documents.
   - Missing source workspace metadata blocks approval rather than being
     silently inferred.

4. Audit the controller formula query path.

   Inspect how graph.v2/controller formula steps find ready work, verify
   dependencies, claim routed work, and read metadata under a DoltLite-backed
   store. Evidence should cite specific files, tests, and commands. Treat broad
   scans, stale cached reads, or provider-specific assumptions outside the beads
   boundary as findings.

5. Audit the controller formula write path.

   Inspect how graph.v2/controller formula steps create attempt beads, update
   status, record metadata, close work, and propagate outcomes under the
   DoltLite provider. Writes must stay on the provider-owned mutation path and
   should not rely on raw SQL shortcuts, stale status files, or Dolt SQL server
   assumptions.

6. Add focused verification only after evidence identifies the exact gap.

   If current tests already cover a query or write behavior, cite them. If a gap
   is found, add the smallest focused test around the responsible package. Do
   not run the full suite locally, do not restart the live city, and do not
   install or replace live `gc`, `bd`, or `doltlite-client` binaries.

7. Write the final audit artifact.

   Write `query-write-audit.md` under this artifact root. The report should map
   each audited query/write behavior to source evidence, focused verification,
   outcome, remaining risk, and follow-up work. Update the manifest with the
   report entry and the latest source metadata.

## Non-Goals

- Do not implement production source changes as part of this plan-writing step.
- Do not infer source identity from document workspace change IDs.
- Do not copy document bodies into bead metadata.
- Do not run broad `bd ready`, root-bead searches, or unrelated workflow
  discovery from worker sessions.
- Do not run the full Go test suite locally.
- Do not restart the live city, kill tmux servers, or replace live binaries.
- Do not introduce role-specific logic into Go to make this audit pass.

## Verification

For this plan artifact:

- Confirm frontmatter declares `schema: gc.build.plan.v1`.
- Confirm required sections are present: Summary, Current System, Proposed
  Implementation, Non-Goals, and Verification.
- Recompute the SHA-256 hash of `plan.md` and record it in `manifest.json`.
- Record the current jj document change ID separately from any source change ID.

For the downstream implementation convoy:

- Run focused controller/formula tests that cover graph.v2 query and write
  behavior once the exact source surfaces are known.
- Run focused DoltLite store tests around `internal/beads/doltlite_read_store.go`
  and `cmd/gc/doltlite_store_native.go` if source changes touch those paths.
- Run focused pack or diagnostic tests under `examples/beads-doltlite/` and
  `tools/doltlite-client/` only if the audit touches provider diagnostics.
- Record every command as run, skipped, or blocked with the concrete reason.

This workflow remains blocked for approval until a real source workspace path
and source change ID are recorded.
