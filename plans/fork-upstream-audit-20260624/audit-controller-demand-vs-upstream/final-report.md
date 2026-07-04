---
schema: gc.build.final-report.v1
workflow:
  id: gc-b7tg
  formula: jj-build
methodology:
  pack: gascity-jj-base
  name: jj-managed build-base
producer:
  formula: jj-build
  stage: finalize.iteration.1
  attempt: 1
status: approved
trace:
  upstream:
    - path: plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/requirements.md
      hash: sha256:3acc0e95ba45ba99a38b66ee92f358e6e0b8959e4214f9e29e0047ee1833e763
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
        - REQ-005
        - REQ-006
        - REQ-007
    - path: plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/plan.md
      hash: sha256:6f0c6911a55ed6f079b60fd1837747bca4beefe89b34fbd917766956a9a315fd
    - path: plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/decomposition.md
      hash: sha256:e245c72107094d6c4d035bd2e95ff1abb3b66585bea979ac03aa188f42980fe2
    - path: plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/implementation-summary.md
      hash: sha256:29a63d3782bd96599f1c1cc986f0faa6df0578bb4e4109cace03763550390e08
    - path: plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/plan-review.md
      hash: sha256:0387fc700fd0773fde4ed5974a76692be2028f2a27632c54e8751dc31efdd9d8
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
---

# Final Report: Controller Demand vs Upstream Audit

## Summary

This jj-managed build completed the document-only audit for the controller
demand failure. The approved artifacts show that direct discovery can see
normal worker-routed work while the controller demand trace can still report
only control-dispatcher routes. The implementation stayed within the approved
scope: it produced evidence and a follow-up target, not a source fix.

Continuation entrypoint: `jj-build` on workflow root `gc-b7tg`.

Skipped or reused stages: the finalize stage reused the manifest-managed
requirements, plan, plan review, decomposition, implementation summary, and
review artifacts already produced for this run. No repair loop ran, and no
publish action is authorized because the workflow variables have
`push=false` and `open_pr=false`.

## Outcome

Status: approved.

Review verdict: pass.

| ID | Status |
| --- | --- |
| REQ-001 | covered |
| REQ-002 | covered |
| REQ-003 | covered |
| REQ-004 | covered |
| REQ-005 | covered |
| REQ-006 | covered |
| REQ-007 | covered |

Implementation evidence is present. The source workspace is
`/data/projects/doltlite-gascity/gascity/.gc/workspaces/gascity/packs/gascity-jj-base`,
workspace name `gascity-jj-base`, at source change
`yqwwsuuurskrqytnovpqzstsmyywnmml`. The source change is described as
`audit: compare controller demand against upstream` and the review confirmed it
does not carry source-file changes.

The audit handoff identifies a focused next implementation target: add or
extend a `cmd/gc` demand regression test for normal worker-routed demand before
changing controller demand behavior. The existing nearby surfaces are
`cmd/gc/scale_from_zero_no_scalecheck_test.go` and
`cmd/gc/scale_from_zero_named_no_scalecheck_test.go`.

Next action: hand off the approved final report and manifest. Publish should
no-op unless a later workflow explicitly enables push or PR creation.

## Artifacts

| Artifact | Schema | Path | Change ID | Notes |
| --- | --- | --- | --- | --- |
| Requirements | `gc.build.requirements.v1` | `plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/requirements.md` | `mxpuqnnzrwsttwtokxkrvwpzwoytozwx` | Defines REQ-001 through REQ-007 and the document-only scope. |
| Plan | `gc.build.plan.v1` | `plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/plan.md` | `vzroqmvr` | Names the controller demand files, DoltLite/cache surfaces, and focused test surfaces. |
| Decomposition | `gc.build.decomposition.v1` | `plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/decomposition.md` | `qrozssznnnywnvknoszozqskmnrypulp` | Keeps the work to a single document-only audit item. |
| Implementation summary | `gc.build.implementation-summary.v1` | `plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/implementation-summary.md` | `mrlrpzyotstssktvtwzlqywoqvwwppyz` | Records source identity, implementation evidence, and requirement coverage. |
| Review | `gc.build.review.v1` | `plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/plan-review.md` | `yvvnxpluprrwzszooxzxsqwzzrqlwksx` | Approves the implementation summary with no blocking findings. |
| Final report | `gc.build.final-report.v1` | `plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/final-report.md` | `qlnsvzmqvlunlzvylklsuuryxqsvvlxn` | This document; the manifest records its final hash. |

Implementation convoy: `gc-wbd5`.

Manifest:
`/data/projects/doltlite-gascity/gascity/plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/manifest.json`.

## Remaining Risks

No blocking risks remain for this document workflow.

Open follow-up risk: the controller demand bug itself is not fixed by this
audit. The next implementation should prove or update the recommended
regression target before changing `defaultScaleCheckCountsAndDemand`,
`readyForControllerDemand`, demand snapshot refresh, desired-state merging, or
DoltLite/cache ready-read behavior.

No unresolved decisions.
