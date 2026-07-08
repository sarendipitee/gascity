---
schema: gc.build.plan.v1
workflow:
  id: gc-514m
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
    - path: plans/gc-vzbu/requirements.md
      hash: sha256:5bf37714d71c252d2f82bc63aaeea4e40c30feae1db357917c12894bee8498da
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

# Implementation Plan: Clean DoltLite Gas City Build Workspace

## Summary

Create a dedicated Gas City jj workspace for DoltLite-linked `gc` builds and
make beads-doltlite build/install commands use that workspace explicitly. Add a
preflight that fails before install when the source workspace is missing, on the
wrong line, or dirty in a way that makes the build source ambiguous. Keep all
workflow documents and manifest updates in `plans/gc-vzbu`.

## Current System

- The beads-doltlite build script resolves the Gas City source from
  `$CITY_ROOT/gascity` unless `--gc-source` or `GASCITY_SRC` is supplied.
- The default checkout at `/data/projects/doltlite-gascity/gascity` is used for
  PR review and workflow documents, so it can sit on PR or document changes that
  are not the intended DoltLite integration line.
- The intended integration source is represented by bookmark
  `gc-0a4-doltlite-lock-pool-fix` and change
  `vorpomlsorrwykqxnuxvtzutrvrzpqxu`.
- Existing installed-build tags must remain historical evidence only; they do
  not prove that a clean intended source revision was installed.

## Proposed Implementation

1. Establish `/data/projects/doltlite-gascity/gascity-build` as the build
   workspace.
   - If it already exists, verify it is a jj workspace for the Gas City repo.
   - If it is missing, create it as a jj workspace from the same repository and
     position it on `gc-0a4-doltlite-lock-pool-fix`.
   - Do not reset, delete, or rewrite the default checkout or the
     `pr/runtime-ready-demand-snapshot` line.

2. Add a build-source preflight to the beads-doltlite build path.
   - Resolve the Gas City source from explicit `--gc-source` first, then
     `GASCITY_SRC`, and fail before install when neither points to the approved
     build workspace.
   - Run `jj -R "$GC_SOURCE" status` and require no unrelated working-copy
     changes.
   - Run `jj -R "$GC_SOURCE" log -r @ --no-graph` and require the selected
     change to be on or descendant from the intended DoltLite integration line.
   - Print the selected source path and revision before any install step.

3. Document the approved install command.
   - Use an explicit Gas City source path:
     `gc beads-doltlite build all --install --gc-source /data/projects/doltlite-gascity/gascity-build`.
   - Include the relevant beads source behavior only where it affects the
     install command.
   - State that `/data/projects/doltlite-gascity/gascity` is the document and
     PR checkout, not the default install source for DoltLite-linked builds.

4. Preserve installed-build traceability.
   - Create installed-build tags or labels only after an install succeeds.
   - Record the exact source path, jj change ID, and commit hash that produced
     the installed binary.
   - Do not reuse or reinterpret old `build/gc-installed-*` tags as proof for
     this build.

## Non-Goals

- Do not rewrite, reset, or delete the default PR/docs checkout.
- Do not change the upstream PR branch `pr/runtime-ready-demand-snapshot`.
- Do not redesign the beads-doltlite build system beyond explicit source
  selection, preflight validation, runbook clarity, and installed-build
  traceability.
- Do not push changes or open a PR from this workflow run.

## Verification

- Verify workspace state with:
  `jj -R /data/projects/doltlite-gascity/gascity-build status`.
- Verify integration-line ancestry with:
  `jj -R /data/projects/doltlite-gascity/gascity-build log -r @ --no-graph`.
- Exercise the preflight with the correct `--gc-source` and confirm it prints
  the selected source path and revision before build/install.
- Exercise the preflight with `/data/projects/doltlite-gascity/gascity` and
  confirm it fails before install while that checkout is on PR or document work.
- Run a no-install dry build or bounded build check when the implementation
  supports it; otherwise record why a full install was intentionally deferred.

## Rollout

1. Land the workspace/runbook/preflight changes without pushing or opening a PR
   from this workflow.
2. Run the validation commands and capture the `jj status` and `jj log`
   evidence for the selected build workspace.
3. Only after validation passes, run the explicit install command from the
   approved build workspace.
4. Tag or label the installed revision after install succeeds.

## Open Questions

- Should the preflight create `/data/projects/doltlite-gascity/gascity-build`
  automatically when missing, or fail with operator instructions?
- Should integration-line validation require the bookmark exactly at `@`, or
  accept descendants of `gc-0a4-doltlite-lock-pool-fix`?
- What bounded build check should be used when a full install is too expensive
  for a workflow validation lane?
