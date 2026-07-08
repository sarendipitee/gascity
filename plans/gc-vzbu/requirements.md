---
schema: gc.build.requirements.v1
workflow:
  id: gc-514m
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
    - path: beads/gc-vzbu
      hash: bead:gc-vzbu
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
        - REQ-005
        - REQ-006
        - REQ-007
        - REQ-008
    - path: plans/gc-vzbu/context.yaml
      hash: sha256:38b12c4241d69ed3f9ffe88129ae02d546ec6be3f9531b2d0645e9034fcb5266
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
        - REQ-005
        - REQ-006
        - REQ-007
        - REQ-008
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
    - id: REQ-007
      status: covered
    - id: REQ-008
      status: covered
---

# Requirements: Clean DoltLite Gas City Build Workspace

## Problem Statement

The beads-doltlite pack can build and install `gc` from the live Gas City source checkout discovered at `$CITY_ROOT/gascity` unless the operator supplies an explicit source path. The current default checkout is used for PR and document workflow work, so installing from it can produce a binary that does not include the intended local DoltLite integration changes.

The build workflow needs a clean, dedicated Gas City source workspace for DoltLite-linked builds. It must keep install/build state separate from PR review and documentation state, make the intended source path explicit, and fail before install when the source workspace is not the intended integration line.

## W6H

| Question | Answer |
| --- | --- |
| Who | Operators and agents running beads-doltlite build/install workflows for this Gas City rig. |
| What | Use a dedicated Gas City jj workspace as the build source for DoltLite-linked `gc` builds. |
| When | Before any `gc beads-doltlite build all --install` run that should include local DoltLite integration work. |
| Where | Expected build workspace: `/data/projects/doltlite-gascity/gascity-build`; default document and PR checkout: `/data/projects/doltlite-gascity/gascity`. |
| Why | Prevent installs from the PR/docs checkout and keep installed binaries traceable to the source revision that actually built them. |
| How | Require an explicit build source path, verify the source workspace state before install, and document the approved invocation. |
| How much | Cover the local rig workflow and build runbook path; do not redesign jj, beads, or the build system globally. |

## User Stories

1. As a maintainer running a DoltLite-linked install, I need the build source to be an explicit clean workspace so the installed `gc` binary includes the intended integration changes.
   - The workflow identifies the source workspace path before build.
   - The source workspace is separate from the default PR/docs checkout.
   - The build command does not silently fall back to `/data/projects/doltlite-gascity/gascity` when that checkout is on PR or document work.

2. As an operator, I need a documented build invocation so repeated installs use the same source checkout.
   - The runbook or command uses an explicit `--gc-source /data/projects/doltlite-gascity/gascity-build` path or an equivalent documented input.
   - Relevant `--bd-source` or environment-variable behavior is documented where it affects the install.
   - The documented command remains compatible with `gc beads-doltlite build all --install` once the source path is correct.

3. As a reviewer, I need evidence that the build source is on the intended DoltLite integration line.
   - Validation records `jj status` and `jj log` evidence for the build workspace.
   - The evidence distinguishes the DoltLite integration line from `pr/runtime-ready-demand-snapshot` and document-workflow changes.
   - The implementation does not delete, reset, or rewrite existing PR or integration history.

## Technical Stories

1. Define or reuse a dedicated jj workspace for build/install source.
   - Expected name/path: `gascity-build` at `/data/projects/doltlite-gascity/gascity-build`.
   - The workspace must be positioned on the intended DoltLite integration line, represented by bookmark `gc-0a4-doltlite-lock-pool-fix` and change `vorpomlsorrwykqxnuxvtzutrvrzpqxu`.
   - If an existing workspace is reused instead, the implementation must document why it is the canonical build workspace.

2. Make the beads-doltlite build source explicit.
   - The build path must accept or document an explicit `--gc-source` value.
   - The selected source path must be visible in the runbook or command output used for install.
   - The build should record the source revision that produced the installed binary.

3. Add preflight protection for accidental installs from the wrong checkout.
   - The preflight fails before install when the source workspace is missing.
   - The preflight fails before install when the source workspace is on the PR/docs line instead of the intended integration line.
   - The preflight fails before install when unrelated working-copy changes make the build source ambiguous.

4. Preserve traceability for installed builds.
   - Build tags or installed-build labels are created only for revisions that actually produced installed binaries.
   - Existing tags such as `build/gc-installed-20260623T013004Z-dirty-base` must not be treated as proof that a clean intended source was installed.

## Behavior Requirements

| ID | Requirement |
| --- | --- |
| REQ-001 | A dedicated Gas City build workspace must exist at `/data/projects/doltlite-gascity/gascity-build`, or the implementation must document a deliberate equivalent workspace. |
| REQ-002 | The build workspace must be on the intended DoltLite integration line, not the `pr/runtime-ready-demand-snapshot` PR/docs line. |
| REQ-003 | The documented install command must use an explicit Gas City source path, such as `--gc-source /data/projects/doltlite-gascity/gascity-build`, or an equivalent documented input. |
| REQ-004 | The build/install path must fail before install if the source workspace is missing, on the wrong line, or has unrelated changes. |
| REQ-005 | Installed-build tags or labels must only mark source revisions that actually produced installed binaries. |
| REQ-006 | The implementation must preserve current jj history and must not delete, reset, or rewrite `pr/runtime-ready-demand-snapshot` or the DoltLite integration line. |
| REQ-007 | Validation must include `jj status` and `jj log` evidence for the selected source workspace plus a no-install dry build or bounded build check when appropriate. |
| REQ-008 | Workflow documents and manifest metadata must remain under `plans/gc-vzbu` in the default@ document workspace, separate from build/install state. |

## Example Mapping

| Input | Requirement | Expected Output |
| --- | --- | --- |
| `gc-vzbu` asks for a clean DoltLite Gas City build workspace. | REQ-001, REQ-002 | A dedicated `gascity-build` jj workspace or documented equivalent is selected as the build source. |
| beads-doltlite currently resolves `$CITY_ROOT/gascity` unless overridden. | REQ-003, REQ-004 | The build/install invocation uses an explicit source path and preflight checks reject accidental fallback. |
| Existing PR/docs work lives in the default checkout. | REQ-006, REQ-008 | PR/document work remains isolated from build source state. |
| Installed build tags should reflect real installed binaries. | REQ-005, REQ-007 | Validation evidence and build tags map to the actual source revision used for install. |

## Acceptance Criteria

| ID | Status |
| --- | --- |
| REQ-001 | covered |
| REQ-002 | covered |
| REQ-003 | covered |
| REQ-004 | covered |
| REQ-005 | covered |
| REQ-006 | covered |
| REQ-007 | covered |
| REQ-008 | covered |

- A dedicated build workspace exists, or reuse of an existing workspace is explicitly justified.
- `jj status` and `jj log` evidence shows the build workspace is on the intended DoltLite integration line.
- The documented beads-doltlite build command uses an explicit Gas City source path and covers relevant beads source behavior.
- A preflight or runbook prevents accidental install from the default PR/docs checkout.
- Validation includes either a no-install dry build or a bounded build check when appropriate for the implementation.
- No push or PR is performed unless a later approved workflow explicitly asks for it.

## Out Of Scope

- Rewriting the upstream PR branch `pr/runtime-ready-demand-snapshot`.
- Deleting or resetting jj history to force a workspace into shape.
- Redesigning the beads-doltlite build system beyond source-path selection, preflight validation, and runbook clarity.
- Opening or pushing a PR from this workflow run.
- Replacing the default@ document workspace or moving workflow artifacts outside `plans/gc-vzbu`.

## Open Questions

- Should the build workspace be created by the workflow when missing, or should the implementation fail with operator instructions?
- What exact bounded build check is acceptable if a full install is too expensive during validation?
- Should the preflight key off bookmark ancestry, an explicit allowlist of change IDs, or both?
