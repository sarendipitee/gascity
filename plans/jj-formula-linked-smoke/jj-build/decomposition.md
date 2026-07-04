---
schema: gc.build.decomposition.v1
workflow:
  id: gc-wrkw
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
    - path: /data/projects/doltlite-gascity/gascity/plans/jj-formula-linked-smoke/jj-build/requirements.md
      hash: sha256:0be5778c2c7279867f647a7a4c7e2ab9c056b6d7d04fe805ef8ed258ebc8d4d5
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
        - REQ-005
        - REQ-006
    - path: beads/gc-wrkw
      hash: bead:gc-wrkw
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

# Decomposition: Linked DoltLite JJ Build Smoke

## Summary

This decomposition keeps the `jj-build` smoke intentionally document-scoped.
The approved requirements ask the workflow to prove that a linked DoltLite
bead store can drive `jj-build` while durable build documents remain normal
jj-tracked files under `plans/jj-formula-linked-smoke/jj-build`.

The implementation convoy therefore contains a single verification work item.
It does not request source changes, pull-request work, or a new formula shape.
The work item verifies that downstream stages can continue from manifest-managed
paths, schema IDs, hashes, and jj change IDs.

Coverage matrix:

| ID | Status |
| --- | --- |
| REQ-001 | covered |
| REQ-002 | covered |
| REQ-003 | covered |
| REQ-004 | covered |
| REQ-005 | covered |
| REQ-006 | covered |

## Selected Downstream Formulas

| Formula | Target | Input | Purpose |
| --- | --- | --- | --- |
| jj-implement | gc.implementation-worker | gc-l444 | Drain the manifest-handoff verification convoy. |
| jj-do-work-item | gc.implementation-worker | gc-8j23 | Execute the single document-only smoke verification item. |

## Implementation Convoy

- Convoy ID: `gc-l444`
- Convoy name: `jj-build-manifest-handoff-implementation`
- Work items: `gc-8j23`
- Document workspace: `default`
- Artifact root: `plans/jj-formula-linked-smoke/jj-build`
- Manifest: `plans/jj-formula-linked-smoke/jj-build/manifest.json`

## Work Items

### gc-8j23: Verify jj-build manifest-backed handoff

Verify the `jj-build` smoke handoff without source changes.

- Confirm workflow root `gc-wrkw` remains tied to source bead `gc-l8gl`.
- Confirm documents live under `plans/jj-formula-linked-smoke/jj-build` in
  `default@`.
- Confirm `manifest.json` records document path, absolute path, schema, SHA-256
  hash, and jj change ID for each produced document.
- Confirm downstream stages consume manifest-managed paths rather than pasted
  document bodies.
- Confirm DoltLite remains the live bead store and bead metadata stores only
  paths, schemas, hashes, and change IDs.
- Confirm validation coverage remains deterministic and repairable.

Coverage: REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006.
