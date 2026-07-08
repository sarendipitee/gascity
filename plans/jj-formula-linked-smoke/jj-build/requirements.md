---
schema: gc.build.requirements.v1
workflow:
  id: gc-wrkw
  formula: jj-build
methodology:
  pack: gascity-jj-base
  name: jj-build
producer:
  formula: jj-build
  stage: requirements
  attempt: 1
status: approved
trace:
  upstream:
    - path: beads/gc-l8gl
      hash: bead:gc-l8gl
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

# Requirements: Linked DoltLite JJ Build Smoke

## Problem Statement

Gas City needs a smoke workflow that proves the `jj-build` graph formula can run
against a linked `beads-doltlite` installation without losing the separation
between live bead state and durable workflow documents. The source bead
`gc-l8gl` exists specifically to test `jj-build` formula sling behavior after a
linked DoltLite install. The smoke must show that the formula can prepare a
default@ document workspace, produce manifest-managed build documents, and keep
downstream stages pointed at concrete file paths instead of prompt-only content.

## W6H

| Question | Answer |
| --- | --- |
| Who | Gas City workflow operators and agents running the `jj-build` formula. |
| What | A linked DoltLite smoke run that produces the build requirements artifact in the default@ document workspace. |
| When | During the `requirements` stage of workflow root `gc-wrkw`, before plan, decomposition, implementation, review, and publish stages. |
| Where | Document files live under `plans/jj-formula-linked-smoke/jj-build`; live work state remains in the DoltLite-backed bead store. |
| Why | To prove the jj-aware build formula preserves Gas City's build-base contract while using manifest-managed documents. |
| How | The formula prepares a manifest, routes document-producing steps to the default@ workspace, records hashes and jj change IDs, and passes manifest paths to downstream stages. |
| How much | The smoke should cover the minimum end-to-end document handoff needed to unblock later formula stages without implementing source changes. |

## User Stories

### REQ-001: Launch `jj-build` from the linked DoltLite smoke bead

As a workflow operator, I want the smoke source bead to drive a real `jj-build`
workflow so I can verify the linked DoltLite install supports formula sling.

Acceptance criteria:

- Workflow root `gc-wrkw` is tied to source bead `gc-l8gl` through the input
  convoy.
- The formula records its document workspace and manifest metadata on the
  workflow root.
- The requirements artifact traces back to `beads/gc-l8gl`.

### REQ-002: Keep workflow documents in default@

As a document-consuming downstream agent, I want build documents written as
normal files under the default@ artifact root so later stages can read stable
paths instead of relying on prompt context.

Acceptance criteria:

- The manifest records `docs_workspace: default` and `docs_base_revset:
  default@`.
- The artifact root is `plans/jj-formula-linked-smoke/jj-build`.
- Generated document paths are repo-relative and remain inside that artifact
  root unless an explicit path variable overrides them.

### REQ-003: Manage requirements through the manifest

As a later build stage, I want the requirements document recorded in
`manifest.json` with schema, hash, and jj change ID so I can validate and reuse
the artifact without reading bead notes.

Acceptance criteria:

- `requirements.md` declares `schema: gc.build.requirements.v1`.
- `manifest.json` includes a `requirements` document entry with path, absolute
  path, schema, SHA-256 hash, and jj change ID.
- The claimed requirements bead records `gc.docs.requirements.*` metadata.
- The workflow root records the latest `gc.docs.change_id`.

### REQ-004: Preserve the inherited build graph contract

As a formula maintainer, I want `jj-build` to keep the inherited build-base
graph shape while overriding only document-producing behavior.

Acceptance criteria:

- Requirements complete before plan and decomposition stages consume the
  manifest.
- Implementation drains receive the document workspace, artifact root, and
  manifest path.
- Review and finalization stages follow manifest document paths rather than
  pasted context.

### REQ-005: Keep DoltLite as the live work store

As a storage maintainer, I want live bead state to remain in DoltLite while
document bodies stay in jj-tracked files.

Acceptance criteria:

- Bead metadata stores only paths, schemas, hashes, and change IDs.
- Document bodies are not copied into bead notes or metadata.
- The smoke does not introduce a parallel persistence path for live work state.

### REQ-006: Make validation and repair deterministic

As a workflow operator, I want artifact validation to identify exactly which
requirements entry failed so repair attempts can update the document in place.

Acceptance criteria:

- Front matter uses the `gc.build.requirements.v1` mapping shape.
- `trace.coverage` and the Markdown coverage table contain the same ID/status
  pairs.
- Coverage statuses use `covered` for satisfied requirements.
- Any unresolved ambiguity is captured in `Open Questions` instead of blocking
  the headless smoke.

## Technical Stories

### REQ-002: Default@ document workspace preparation

The prepare step must resolve the default document workspace to the rig root and
record the root manifest path before requirements are written.

### REQ-003: Manifest-backed artifact handoff

Document-producing steps must update the manifest and bead metadata after
writing each file so downstream graph stages can use structured paths and
hashes.

### REQ-004: JJ-aware build graph continuity

The requirements stage must leave enough manifest state for plan, review,
decomposition, implementation summary, final report, and publish stages to
continue without switching back to prompt-only handoff.

## Behavior Requirements

| ID | Requirement | Status |
| --- | --- | --- |
| REQ-001 | The smoke run is rooted in `gc-l8gl` and executed through `jj-build`. | required |
| REQ-002 | Workflow documents are written under the default@ artifact root. | required |
| REQ-003 | Requirements are recorded in the manifest with schema, hash, and change ID. | required |
| REQ-004 | Downstream build graph stages consume manifest-managed paths. | required |
| REQ-005 | DoltLite remains the live bead store; documents remain jj-tracked files. | required |
| REQ-006 | Validation coverage is deterministic and repairable. | required |

## Example Mapping

| Input | Output |
| --- | --- |
| `beads/gc-l8gl` | `plans/jj-formula-linked-smoke/jj-build/requirements.md` |
| `gc.docs.manifest_path` | `plans/jj-formula-linked-smoke/jj-build/manifest.json` |
| `gc.docs.workspace=default` | document edits occur in `/data/projects/doltlite-gascity/gascity` |
| `gc.build.artifact_schema=gc.build.requirements.v1` | requirements front matter and coverage table use the requirements schema contract |

## Acceptance Criteria

| ID | Status |
| --- | --- |
| REQ-001 | covered |
| REQ-002 | covered |
| REQ-003 | covered |
| REQ-004 | covered |
| REQ-005 | covered |
| REQ-006 | covered |

## Out Of Scope

- Implementing source changes for DoltLite, Beads, or Gas City.
- Opening or pushing a pull request.
- Replacing the inherited build-base graph with a new formula shape.
- Storing full document bodies in beads or workflow metadata.

## Open Questions

- No blocking open questions. Later stages should validate the manifest-backed
  handoff with the files produced by this smoke workflow.
