---
schema: gc.build.final-report.v1
workflow:
  id: gc-514m
  formula: jj-build
methodology:
  pack: gascity-jj-base
  name: jj-build
producer:
  formula: jj-build
  stage: finalize
  attempt: 1
status: blocked
trace:
  upstream:
    - path: /data/projects/doltlite-gascity/gascity/plans/gc-vzbu/requirements.md
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
    - path: /data/projects/doltlite-gascity/gascity/plans/gc-vzbu/implementation-summary.md
      hash: sha256:b5901d2cba78ba675ba129962abe5d69d637ac5212163d4ff1aa9c5b20ab9619
    - path: beads/gc-514m
      hash: bead:gc-514m
  coverage:
    - id: REQ-001
      status: blocked
      rationale: Workflow root gc-514m was closed as a misrouted workflow latch.
    - id: REQ-002
      status: blocked
      rationale: Workflow root gc-514m was closed as a misrouted workflow latch.
    - id: REQ-003
      status: blocked
      rationale: Workflow root gc-514m was closed as a misrouted workflow latch.
    - id: REQ-004
      status: blocked
      rationale: Workflow root gc-514m was closed as a misrouted workflow latch.
    - id: REQ-005
      status: blocked
      rationale: Workflow root gc-514m was closed as a misrouted workflow latch.
    - id: REQ-006
      status: blocked
      rationale: Workflow root gc-514m was closed as a misrouted workflow latch.
    - id: REQ-007
      status: blocked
      rationale: Workflow root gc-514m was closed as a misrouted workflow latch.
    - id: REQ-008
      status: blocked
      rationale: Workflow root gc-514m was closed as a misrouted workflow latch.
---

# Final Report: GC-VZBU JJ Build

## Summary

The `jj-build` run for `gc-514m` ended blocked. The workflow latch was routed to
a normal run-operator worker and closed as a control-plane routing failure before
implementation completed. A schema-valid blocked implementation summary exists
at `plans/gc-vzbu/implementation-summary.md`.

| ID | Status |
| --- | --- |
| REQ-001 | blocked |
| REQ-002 | blocked |
| REQ-003 | blocked |
| REQ-004 | blocked |
| REQ-005 | blocked |
| REQ-006 | blocked |
| REQ-007 | blocked |
| REQ-008 | blocked |

## Outcome

Final outcome: blocked. No source workspace was selected, no install/build proof
was recorded, no review lanes ran, and no publish step ran. The next action is a
control-plane follow-up for the misrouted workflow latch, then a relaunch or
resume of executable child steps.

## Artifacts

- Requirements: `plans/gc-vzbu/requirements.md`
- Context bundle: `plans/gc-vzbu/context.yaml`
- Implementation summary: `plans/gc-vzbu/implementation-summary.md`
- Final report: `plans/gc-vzbu/final-report.md`
- Manifest: `plans/gc-vzbu/manifest.json`
- Implementation convoy: not available from completed child evidence.
- Publish outcome: not run.

## Remaining Risks

The build workspace, explicit source selection, install preflight, tag integrity,
and validation requirements remain unresolved. This final report is a blocked
record, not an approval to install or publish binaries.
