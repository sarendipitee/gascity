---
schema: gc.build.requirements.v1
workflow:
  id: gc-b7tg
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
    - path: beads/gc-b7tg
      hash: bead:gc-b7tg
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
        - REQ-005
        - REQ-006
        - REQ-007
    - path: plans/fork-upstream-audit-20260624/audit.md
      hash: sha256:729518036dfcd328179eabdac3751246b9fbcaeff110cd06473c553e7c3586aa
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
        - REQ-005
        - REQ-006
        - REQ-007
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

# Requirements: Controller Demand vs Upstream Audit

## Problem Statement

The fork-upstream audit identified a controller demand failure in the local Gas
City fork. Direct work discovery paths, including `bd ready` and `gc hook`, can
see worker-routed work, but controller trace output for `scale_check_counts`
only reports control-dispatcher routes. Demand-driven normal workers can remain
idle even when routed work exists.

The suspected divergence is in the local DoltLite/controller line compared with
upstream `main`, especially around `defaultScaleCheckCountsAndDemand`,
`readyForControllerDemand`, runtime demand snapshots, and pool desired-state
merging. The audit needs evidence and a narrow follow-up target before any
source fix, upstream reconciliation, or cleanup happens.

## Solution

Produce an evidence-backed audit that compares the local fork against upstream
`main` for controller demand behavior and records what must be fixed or tested
next. The audit must preserve local work, use jj read-only inspection for
history comparisons, keep the generated document in the default@ artifact root,
and hand off only manifest-managed paths, schemas, hashes, and jj change IDs.

## W6H

- Who: Gas City maintainers reconciling the DoltLite fork with upstream.
- What: Audit why controller demand counts omit normal worker-routed work.
- When: Before source repair, rebase, restore, cleanup, or upstream merge work.
- Where: The Gas City repo at `/data/projects/doltlite-gascity/gascity`,
  especially `cmd/gc/build_desired_state.go`, `cmd/gc/city_runtime.go`, and
  related desired-state tests.
- Why: A controller that misses normal worker demand stalls graph workflows even
  when the bead store contains ready routed work.
- How: Compare local and upstream behavior, preserve evidence, and identify
  focused regression coverage for the failing demand path.

## User Stories

### REQ-001: Prove the controller demand symptom

As a workflow operator, I need the audit to show whether the controller demand
path observes normal worker-routed beads, so that idle workers are tied to
evidence instead of assumptions.

Acceptance criteria:

- The audit records the observed direct-discovery path, such as `bd ready` or
  `gc hook`, that can see the normal worker-routed bead.
- The audit records the controller trace field that misses the same demand,
  including `scale_check_counts` when available.
- The audit distinguishes normal worker routes from control-dispatcher routes.

### REQ-002: Compare the fork against upstream demand semantics

As a maintainer, I need the local fork compared to upstream `main` for the
controller demand surfaces, so that future work can preserve useful fork changes
while identifying the actual divergence.

Acceptance criteria:

- The audit covers `defaultScaleCheckCountsAndDemand`.
- The audit covers `readyForControllerDemand`.
- The audit covers runtime demand snapshot loading, refresh, and fingerprinting
  in `cmd/gc/city_runtime.go`.
- The audit covers pool desired-state merging for demand-driven sessions.
- The audit names any upstream behavior that the fork changed, removed, or
  replaced.

### REQ-003: Identify regression coverage for normal worker demand

As an implementer, I need a precise test target for normal worker-routed demand,
so that a later fix cannot pass by testing only control-dispatcher routes.

Acceptance criteria:

- The audit identifies existing tests that already cover controller demand.
- The audit identifies the missing normal-worker route scenario when no current
  test covers it.
- The expected assertion includes controller desired state or
  `scale_check_counts` reflecting normal worker demand.
- The test target stays focused enough to run without the full test suite.

### REQ-004: Preserve fork state during audit

As the owner of this integration branch, I need the audit to avoid destructive
repository operations, so that upstream reconciliation does not erase local
DoltLite or Gas City integration work.

Acceptance criteria:

- The audit does not run `jj rebase`, `jj restore`, `jj abandon`,
  `git checkout`, `git reset`, or cleanup commands.
- Runtime files, generated artifacts, and bead state are classified as evidence
  unless a later approved task says otherwise.
- Any source comparison uses read-only jj or file inspection.

### REQ-005: Handoff a useful audit artifact

As a reviewer, I need the resulting audit handoff to name files, commands,
observed behavior, and unanswered questions, so that implementation planning can
start without redoing discovery.

Acceptance criteria:

- The audit names the involved files and functions.
- The audit records the upstream baseline revision used for comparison.
- The audit separates confirmed facts from hypotheses.
- The audit lists follow-up work without creating implementation beads itself.

## Technical Stories

### REQ-006: Account for DoltLite read-path parity

As a maintainer, I need the audit to account for DoltLite native read behavior,
because divergence between native reads and CLI `bd ready` can produce the same
class of controller-demand failure.

Acceptance criteria:

- The audit states whether the failing path uses native DoltLite reads, CLI bead
  reads, or both.
- The audit notes whether `internal/beads/doltlite_read_store.go`,
  `internal/beads/bdstore.go`, or `internal/beads/doltlite_count.go` are
  implicated by the observed behavior.
- The audit avoids expanding into a full DoltLite store rewrite.

### REQ-007: Account for divergent runtime-demand history

As a maintainer, I need the audit to note divergent runtime-demand variants, so
that future reconciliation chooses the revision matching the observed behavior.

Acceptance criteria:

- The audit records that `pr/runtime-ready-demand-snapshot` has divergent local
  and origin variants when relevant to the demand snapshot path.
- The audit states which variant, if any, matches the running behavior observed
  during the audit.
- The audit does not force-push, rewrite, or reconcile those variants.

## Behavior Requirements

- Controller demand must be evaluated from the same ready-work facts that
  normal worker claim paths use.
- A controller trace that reports only control-dispatcher templates is not
  sufficient when normal worker-routed beads are ready.
- Demand snapshot reuse must not hide newly ready worker-routed beads.
- Partial demand reads must fail closed rather than causing stale or incomplete
  desired state.
- Route behavior must remain configuration-driven; no source change may add
  hardcoded role names.
- The audit artifact must live under the default@ artifact root and be recorded
  in `manifest.json`.

## Example Mapping

| Evidence | Requirement |
| --- | --- |
| `scale_check_counts` omits normal worker templates | REQ-001, REQ-003 |
| `defaultScaleCheckCountsAndDemand` and `readyForControllerDemand` changed locally | REQ-002 |
| Runtime demand snapshot refresh and fingerprinting changed locally | REQ-002, REQ-007 |
| DoltLite native read path is a high-risk divergence | REQ-006 |
| Fork audit preservation rules prohibit destructive cleanup | REQ-004 |

## Acceptance Criteria

- A later audit document can demonstrate the controller-demand symptom or state
  why it no longer reproduces.
- The comparison against upstream `main` names the exact local deltas that
  affect demand counting, snapshot refresh, or desired-state merge behavior.
- The audit identifies focused regression coverage for normal worker demand.
- The audit keeps source changes, cleanup, and upstream reconciliation out of
  scope.
- The document path, schema, hash, and jj change ID are recorded in the
  workflow manifest and bead metadata.

## Out Of Scope

- Implementing the controller-demand fix.
- Rewriting DoltLite native read behavior beyond noting parity evidence.
- Rebasing, restoring, abandoning, or otherwise rewriting jj history.
- Cleaning generated artifacts, runtime files, or bead state.
- Creating implementation beads or launching follow-up formulas.
- Opening or pushing a pull request.

## Other Notes

- Upstream baseline from the parent audit: `pmkksnuywmkw e22049f86666`,
  `Fail closed pool creates on partial scale_check reads (#3686)`.
- Current audit focus files include `cmd/gc/build_desired_state.go`,
  `cmd/gc/build_desired_state_test.go`, `cmd/gc/city_runtime.go`, and
  `cmd/gc/pool_desired_state_test.go`.
- The parent audit also flags DoltLite native store parity and divergent
  `pr/runtime-ready-demand-snapshot` revisions as related risks.

## Open Questions

- Is the missing demand caused by native DoltLite ready-work reads,
  desired-state merging, demand snapshot staleness, or a combination?
- Which divergent `pr/runtime-ready-demand-snapshot` revision matches the
  currently running controller behavior?
- Which focused test should become the durable regression guard for normal
  worker-routed demand?
