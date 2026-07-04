---
schema: gc.build.plan.v1
workflow:
  id: gc-wrkw
  formula: jj-build
methodology:
  pack: gascity-jj-base
  name: jj-build
producer:
  formula: jj-build
  stage: plan
  attempt: 1
status: approved
trace:
  upstream:
    - path: plans/jj-formula-linked-smoke/jj-build/requirements.md
      hash: sha256:0be5778c2c7279867f647a7a4c7e2ab9c056b6d7d04fe805ef8ed258ebc8d4d5
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
        - REQ-005
        - REQ-006
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

# Implementation Plan: Linked DoltLite JJ Build Smoke

## Summary

Verify that the `jj-build` formula can run from the linked DoltLite smoke bead
while keeping live work state in DoltLite and managed build documents in the
`default@` jj workspace. The implementation scope is document-flow validation,
not source changes.

The plan continues from `requirements.md` and should produce enough manifest
state for downstream decomposition, implementation summary, review, and
finalization steps to consume paths, schemas, hashes, and jj change IDs without
copying document bodies into bead metadata.

## Current System

The smoke workflow root is `gc-wrkw`. Its artifact root is
`plans/jj-formula-linked-smoke/jj-build`, and `manifest.json` already records
the `default` document workspace, `default@` base revset, and managed entries
for `requirements`, `decomposition`, `implementation-summary`, and
`final-report`.

The requirements establish six covered constraints:

| ID | Requirement | Status |
| --- | --- | --- |
| REQ-001 | The smoke run is rooted in `gc-l8gl` and executed through `jj-build`. | covered |
| REQ-002 | Workflow documents are written under the default@ artifact root. | covered |
| REQ-003 | Requirements are recorded in the manifest with schema, hash, and change ID. | covered |
| REQ-004 | Downstream build graph stages consume manifest-managed paths. | covered |
| REQ-005 | DoltLite remains the live bead store; documents remain jj-tracked files. | covered |
| REQ-006 | Validation coverage is deterministic and repairable. | covered |

The decomposition has narrowed execution to a single verification item,
`gc-8j23`, which checks the manifest-backed handoff. Existing reports note that
the smoke remains incomplete until that handoff is verified.

## Proposed Implementation

1. Keep the document workspace as `/data/projects/doltlite-gascity/gascity`
   with `docs_workspace=default` and `docs_base_revset=default@`.
2. Treat `plans/jj-formula-linked-smoke/jj-build/manifest.json` as the
   authoritative document handoff. Every document-producing step must record
   `path`, `absolute_path`, `schema`, `hash`, and `change_id`.
3. Write the plan document to
   `plans/jj-formula-linked-smoke/jj-build/plan.md` using
   `gc.build.plan.v1`, then add the `plan` entry to the manifest.
4. Route implementation to the existing decomposition item `gc-8j23`. That
   item should verify that downstream stages can read the manifest, locate the
   requirements and plan artifacts, and compare recorded SHA-256 hashes with
   file contents.
5. Leave source work out of scope unless the verification item finds a concrete
   manifest, path-resolution, or jj-change recording defect.
6. Preserve the live-work boundary: bead metadata stores only paths, schemas,
   hashes, and change IDs; document bodies stay in jj-tracked files.

## Testing

The verification item should perform deterministic checks:

- Parse `manifest.json` and confirm it uses `gascity.workflow-docs.v1`.
- Confirm the manifest contains `requirements` and `plan` document entries.
- Confirm each managed document path is inside
  `plans/jj-formula-linked-smoke/jj-build`.
- Recompute SHA-256 for `requirements.md` and `plan.md` and compare against the
  manifest entries.
- Confirm the plan front matter declares `schema: gc.build.plan.v1`.
- Confirm coverage rows for REQ-001 through REQ-006 are present and marked
  `covered`.

No full local test suite is required for this document-only smoke. If source
changes become necessary, the implementer should run only focused checks for
the touched path.

## Rollout

This is a workflow smoke artifact. After the plan and manifest are updated, the
next graph stage can consume the manifest-managed plan path. The smoke should
remain blocked until `gc-8j23` verifies the handoff and records its result.

## Open Questions

No blocking open questions. If `gc-8j23` finds a mismatch between manifest
metadata and file contents, repair should update the document and manifest in
place rather than creating a parallel persistence path.
