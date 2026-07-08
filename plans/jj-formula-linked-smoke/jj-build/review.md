schema: gc.build.review.v1
workflow:
  id: gc-wrkw
  formula: jj-build
methodology:
  pack: gascity-jj-base
  name: jj-review
producer:
  formula: jj-build
  stage: review
  attempt: 1
status: blocked
verdict: iterate

# Review: Linked DoltLite JJ Build Smoke

## Summary

The review cannot approve the implementation because the manifest and workflow
root do not record a source workspace path or source change ID for source
inspection, and the implementation summary says the verification item has not
completed.

This report uses the manifest as the source of truth and does not inspect the
default document workspace as source state. The recorded source fields are
empty in `manifest.json`, while the root bead metadata has
`gc.docs.source_change_id=none` and no `gc.docs.source_workspace_path`.

## Findings

### Required Fix: Record source workspace/change evidence before source review

Requirement reference:

- REQ-004 and REQ-006 in
  `plans/jj-formula-linked-smoke/jj-build/requirements.md`.

Task reference:

- `gc-8j23` in
  `plans/jj-formula-linked-smoke/jj-build/decomposition.md`.

Evidence:

- `plans/jj-formula-linked-smoke/jj-build/manifest.json` has empty
  `source_workspace_path` and `source_change_id` fields.
- Workflow root `gc-wrkw` metadata has `gc.docs.source_change_id=none` and no
  `gc.docs.source_workspace_path`.
- `plans/jj-formula-linked-smoke/jj-build/implementation-summary.md` records
  REQ-001 through REQ-006 as `blocked`, says implementation convoy `gc-l444`
  remains open, and says verification task `gc-8j23` has not completed.

Impact:

The review lane is required to hard-stop when `gc.docs.source_workspace_path` is
missing or is not a jj workspace. Without source workspace and source change
metadata, it cannot verify that the source state matches the managed document
set or that the smoke's manifest-backed handoff is complete.

Smallest required fix:

Complete `gc-8j23`, record source workspace/change metadata or explicitly record
that the smoke has no source state to inspect, regenerate the implementation
summary, and rerun review against the updated manifest.

## Document Review Notes

- The managed requirements, plan, decomposition, and implementation-summary
  documents are present under `plans/jj-formula-linked-smoke/jj-build`.
- The manifest contains entries for requirements, plan, decomposition,
  implementation-summary, and final-report with schemas, hashes, and jj change
  IDs.
- The review document itself was written under the default document workspace,
  not into bead notes or metadata.
