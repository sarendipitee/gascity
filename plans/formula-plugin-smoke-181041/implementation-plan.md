---
schema: gc.build.plan.v1
workflow:
  id: gc-wisp-b5v
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
    - path: plans/formula-plugin-smoke-181041/requirements.md
      hash: sha256:a74b87556a6e7ffa02323289297f9a1389edcf5307da39f5be43b9bd41901ec8
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
        - REQ-005
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
---

# Implementation Plan: Formula Plugin Smoke

## Summary

Validate the `jj-build` formula-plugin smoke by keeping live workflow state in
the DoltLite-backed bead store while writing durable planning documents into the
`default@` jj document workspace. The scope is document-flow validation only:
no source implementation, push, bookmark, or pull-request side effect is part of
this smoke.

The plan continues from `requirements.md` and gives downstream graph stages a
manifest-managed handoff. Beads should carry paths, schemas, SHA-256 hashes, and
jj change IDs; document bodies remain normal files under
`plans/formula-plugin-smoke-181041`.

## Current System

The workflow root is `gc-wisp-b5v`, running formula `jj-build` from the
`gascity-jj-base` pack. The artifact root is
`plans/formula-plugin-smoke-181041`, and `manifest.json` already records the
`default` document workspace at `/data/projects/doltlite-gascity/gascity`, the
`default@` base revset, and a managed `requirements` document entry.

The approved requirements establish five constraints:

| ID | Requirement | Status |
| --- | --- | --- |
| REQ-001 | Workflow documents live as files in the default jj workspace. | covered |
| REQ-002 | Downstream handoff is manifest-managed. | covered |
| REQ-003 | DoltLite remains the live workflow store. | covered |
| REQ-004 | The smoke avoids source, push, and pull-request side effects. | covered |
| REQ-005 | Downstream build-base stages receive concrete artifact metadata. | covered |

The root variables set `push=false`, `open_pr=false`, and
`drain_policy=separate`. Those values keep this run focused on graph execution
and document handoff instead of remote publication.

## Proposed Implementation

1. Keep the document workspace as `/data/projects/doltlite-gascity/gascity`
   with `docs_workspace=default` and `docs_base_revset=default@`.
2. Treat
   `plans/formula-plugin-smoke-181041/manifest.json` as the authoritative
   handoff between graph stages.
3. Write this plan to
   `plans/formula-plugin-smoke-181041/implementation-plan.md` with schema
   `gc.build.plan.v1`, then add or update the manifest `plan` document entry.
4. Record `path`, `absolute_path`, `schema`, `hash`, and `change_id` for the
   plan entry. Use the jj document change ID that contains both this file and
   the manifest update.
5. Keep the implementation stage document-only. It should verify that the
   manifest can be parsed, that managed paths resolve inside the artifact root,
   and that recorded hashes match file contents.
6. Escalate to source work only if a downstream verification item finds a
   concrete defect in manifest path resolution, jj change recording, or graph
   document handoff. Such source work must use the configured implementation
   formula and a source workspace change ID, not this document change ID.

## Testing

The downstream verification should run deterministic artifact checks:

- Parse `manifest.json` and confirm it uses schema
  `gascity.jj-doc-manifest.v1`.
- Confirm the manifest contains `requirements` and `plan` document entries.
- Confirm both document paths are under
  `plans/formula-plugin-smoke-181041`.
- Recompute SHA-256 for `requirements.md` and `implementation-plan.md` and
  compare each value with the manifest.
- Confirm this plan declares `schema: gc.build.plan.v1`.
- Confirm coverage rows for REQ-001 through REQ-005 are present and marked
  `covered`.

No full local test suite is required for this document-only smoke. If later
source changes are required, the implementer should run only focused checks for
the touched package or command path.

## Rollout

After this plan and the manifest are updated, the next graph stage can consume
the manifest-managed `plan` document path. The smoke remains local: do not push,
open a pull request, or create remote-facing bookmarks for this workflow.

## Open Questions

No blocking open questions. If later graph stages find stale manifest metadata,
repair should update the existing document and manifest entries in place rather
than introducing another persistence path.
