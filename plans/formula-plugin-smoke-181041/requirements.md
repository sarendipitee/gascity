---
schema: gc.build.requirements.v1
workflow:
  id: gc-wisp-b5v
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
    - path: beads/gc-wisp-b5v
      hash: bead:gc-wisp-b5v
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

# Requirements: Formula Plugin Smoke

## Problem Statement

Gas City needs a smoke workflow that proves the `jj-build` graph formula can run
from the DoltLite-backed bead store while keeping generated planning documents
in the `default@` jj document workspace. The workflow root describes a
document-workspace implementation of the build-base contract: live workflow
state remains in DoltLite, document-producing steps write normal files under the
artifact root, and downstream stages receive only manifest-managed paths,
schemas, hashes, and jj change IDs.

This smoke should exercise the formula-plugin path without requiring a source
implementation change, a pull request, or a push to remote. It should leave a
durable requirements artifact that downstream decomposition, implementation, and
review steps can consume through `manifest.json`.

## W6H

| Question | Answer |
| --- | --- |
| Who | Gas City maintainers validating graph.v2 formula execution with the `gascity-jj-base` document workflow. |
| What | Produce manifest-managed planning artifacts for a `jj-build` smoke run while preserving live bead state in DoltLite. |
| When | During the `formula-plugin-smoke-181041` workflow rooted at bead `gc-wisp-b5v`. |
| Where | Under `plans/formula-plugin-smoke-181041` in the default document workspace at `/data/projects/doltlite-gascity/gascity`. |
| Why | To verify that formula-plugin document steps can hand off durable files through the manifest instead of embedding document bodies in bead metadata. |
| How | Use the described jj change for document edits, write requirements as a normal file, update `manifest.json`, and record only paths, schemas, hashes, and change IDs on beads. |
| How much | Cover the smoke workflow contract for document generation and handoff; do not broaden scope into source implementation or PR automation. |

## User Stories

### REQ-001: Keep workflow documents in default@

As a workflow maintainer, I need generated requirements to live as a normal file
in the default jj document workspace so document history is reviewable outside
the live bead database.

Acceptance criteria:

- The requirements document is written under `plans/formula-plugin-smoke-181041`.
- The document is part of the current default@ jj change described for this
  requirements work.
- The bead metadata does not contain the document body.

### REQ-002: Maintain manifest-managed handoff

As a downstream formula step, I need requirements discovery to use the manifest
instead of inspecting freeform bead comments or session logs.

Acceptance criteria:

- `manifest.json` includes a `requirements` document entry.
- The entry records the requirements schema, relative path, absolute path,
  SHA-256 content hash, and jj document change ID.
- The workflow root's latest document change ID is updated to the same jj
  change that contains the requirements and manifest edits.

### REQ-003: Preserve DoltLite as the live workflow store

As a Gas City operator, I need this smoke to keep runtime workflow state in the
DoltLite-backed bead store while storing durable documents in jj.

Acceptance criteria:

- The requirements describe DoltLite as the live bead database and default@ as
  the document workspace.
- No source workspace change ID is treated as a document change ID.
- The document step records only metadata needed for downstream routing and
  artifact validation.

### REQ-004: Avoid source and remote side effects

As a maintainer running a smoke workflow, I need the requirements stage to avoid
source edits, pushes, and pull-request creation.

Acceptance criteria:

- The requirements scope remains document-only.
- The workflow respects `open_pr=false` and `push=false` from the root variables.
- Any later implementation work must be launched through the configured
  implementation formula rather than being performed by this requirements step.

### REQ-005: Support downstream build-base stages

As the graph controller, I need this requirements artifact to be concrete enough
for decomposition, implementation, review, and finalization stages to continue.

Acceptance criteria:

- The document identifies the formula, workflow root bead, artifact root, and
  smoke constraints.
- Requirements are numbered and covered in the YAML trace block.
- The output uses schema `gc.build.requirements.v1`.

## Technical Stories

### TS-001: Requirements artifact generation

Write `plans/formula-plugin-smoke-181041/requirements.md` with front matter that
identifies schema `gc.build.requirements.v1`, workflow root `gc-wisp-b5v`,
formula `jj-build`, producer stage `requirements`, and covered requirement IDs.

### TS-002: Manifest update

Update `plans/formula-plugin-smoke-181041/manifest.json` so downstream steps can
locate and validate the requirements document without reading the document body
from bead metadata.

## Out Of Scope

- Implementing source changes in Gas City or T3 Code.
- Creating or pushing a branch, bookmark, or pull request.
- Changing the formula catalog, graph controller, or beads backend behavior.
- Reworking unrelated manifest-managed plans already present in the workspace.

## Notes

- Root bead: `gc-wisp-b5v`.
- Artifact root: `plans/formula-plugin-smoke-181041`.
- Document workspace: `default@` at `/data/projects/doltlite-gascity/gascity`.
- Runtime variables include `push=false`, `open_pr=false`, and
  `drain_policy=separate`.
