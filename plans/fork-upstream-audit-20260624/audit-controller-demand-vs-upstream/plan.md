---
schema: gc.build.plan.v1
workflow:
  id: gc-b7tg
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

# Implementation Plan: Controller Demand vs Upstream Audit

## Summary

Produce a manifest-managed audit handoff for the controller-demand failure
described in the requirements. The work is document-only: gather read-only
evidence, compare the local fork against the resolved upstream baseline,
identify the narrow regression target, and leave source fixes, cleanup, bead
creation, and PR work out of scope.

The plan should result in an audit document that lets an implementer answer why
normal worker-routed beads are visible to direct discovery (`bd ready` or
`gc hook`) but absent from controller demand traces such as
`scale_check_counts`.

## Current System

The relevant controller-demand path is concentrated in these files:

- `cmd/gc/build_desired_state.go`
  - `defaultScaleCheckCountsAndDemand` builds default scale-check demand.
  - `readyForControllerDemand` is the ready-work query surface that must see
    normal worker-routed beads, not just control-dispatcher routes.
  - `DesiredStateResult.NamedSessionDemand`, pool demand filtering, and partial
    query flags feed the reconciler's desired state.
- `cmd/gc/city_runtime.go`
  - `loadDemandSnapshot` caches demand computation.
  - `readyDemandFingerprint`, session fingerprints, config-change handling, and
    snapshot age decide whether the controller reuses or refreshes demand.
  - Pool desired-state computation records trace data for
    `bead_reconcile.compute_pool_desired`.
- `cmd/gc/scale_from_zero_no_scalecheck_test.go` and
  `cmd/gc/scale_from_zero_named_no_scalecheck_test.go`
  - Existing focused tests cover cross-store cold wake, named-session demand,
    template-routed pool fallback, and non-demand cases.
  - The audit must confirm whether these tests already cover the failing
    normal-worker route or whether a new focused case is needed.
- `internal/beads/caching_store_test.go`
  - Existing cache tests prove stale cached ready rows must fall back to
    authoritative live ready queries.
- `internal/beads/doltlite_read_store.go`, `internal/beads/bdstore.go`, and
  `internal/beads/doltlite_count.go`
  - These are the DoltLite native read surfaces that can diverge from CLI
    `bd ready` and must be classified, not rewritten, in this audit.

The existing fork audit records the observed symptom: direct `bd ready` and
`gc hook` can see worker-routed work, while controller trace
`scale_check_counts` only names control-dispatcher routes. It also identifies
the divergent `pr/runtime-ready-demand-snapshot` bookmark variants:

- local `bda1fd03`: `fix(runtime): refresh demand snapshots for routed work`
- origin `4a6be657`: same subject, divergent content

This workspace currently exposes the comparison baseline as jj bookmark `main`;
the audit should resolve and record its exact revision before comparing files.

## Proposed Implementation

1. Confirm the document-only jj context.

   Run:

   ```bash
   jj -R /data/projects/doltlite-gascity/gascity status
   jj -R /data/projects/doltlite-gascity/gascity log -r @ --no-graph
   ```

   Continue only if `@` is described for this plan/audit document work and the
   working copy contains no unrelated source changes.

2. Resolve the upstream baseline and divergent runtime-demand variants.

   Run read-only jj commands and record their output in the audit:

   ```bash
   jj -R /data/projects/doltlite-gascity/gascity log -r main --no-graph
   jj -R /data/projects/doltlite-gascity/gascity log -r pr/runtime-ready-demand-snapshot --no-graph
   jj -R /data/projects/doltlite-gascity/gascity bookmark list
   ```

   If `main` is not the intended upstream baseline in a later workspace, record
   the resolved replacement and why it was used. Do not rebase, restore,
   abandon, force-push, or reconcile either divergent bookmark variant.

3. Reproduce or restate the controller-demand symptom from live evidence.

   Capture the direct-discovery path that sees a normal worker-routed bead and
   the controller trace that misses it. The audit should include:

   - the bead id or route used as evidence,
   - whether direct discovery came from `gc hook --claim --json`, `bd ready`, or
     an existing trace artifact,
   - the `scale_check_counts` routes observed by the controller,
   - a clear distinction between normal worker routes and
     `core.control-dispatcher` routes.

   If the symptom no longer reproduces, state that explicitly and keep the
   previous audit evidence as historical context rather than silently dropping
   it.

4. Compare local demand code against `main`.

   Use read-only comparisons for these files:

   ```bash
   jj -R /data/projects/doltlite-gascity/gascity diff -r main..@ -- cmd/gc/build_desired_state.go cmd/gc/city_runtime.go
   jj -R /data/projects/doltlite-gascity/gascity diff -r main..@ -- cmd/gc/scale_from_zero_no_scalecheck_test.go cmd/gc/scale_from_zero_named_no_scalecheck_test.go internal/beads/caching_store_test.go
   ```

   The audit should separate confirmed facts from hypotheses for:

   - `defaultScaleCheckCountsAndDemand`,
   - `readyForControllerDemand`,
   - `loadDemandSnapshot`,
   - `readyDemandFingerprint`,
   - pool desired-state merging for demand-driven sessions,
   - the route type that is counted or omitted.

5. Classify the DoltLite native read path.

   Inspect, without changing, these files and any existing focused tests:

   ```bash
   jj -R /data/projects/doltlite-gascity/gascity diff -r main..@ -- internal/beads/doltlite_read_store.go internal/beads/bdstore.go internal/beads/doltlite_count.go
   jj -R /data/projects/doltlite-gascity/gascity diff -r main..@ -- internal/beads/doltlite_read_store_test.go internal/beads/caching_store_test.go
   ```

   The audit should state whether the failing controller path uses native
   DoltLite reads, CLI bead reads, cached ready rows, live ready rows, or a
   combination. If the path cannot be proven from current evidence, record that
   as an unanswered question rather than filling the gap with inference.

6. Identify the focused regression target.

   The preferred follow-up is a narrow test around normal worker-routed demand,
   not a broad full-suite requirement. Candidate targets are:

   - a new or extended case in `cmd/gc/scale_from_zero_no_scalecheck_test.go`
     when the missing behavior is pool/template demand,
   - a new or extended case in
     `cmd/gc/scale_from_zero_named_no_scalecheck_test.go` when the missing
     behavior is named-session demand,
   - a DoltLite/caching test only if the evidence proves the controller is
     reading stale or divergent ready rows.

   Expected assertions should reference desired state or `scale_check_counts`
   containing the normal worker route that direct discovery sees.

7. Write the audit handoff.

   The audit output should be a managed document under
   `plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream`.
   It should include:

   - baseline revisions and commands used,
   - symptom evidence,
   - upstream/local comparison findings,
   - DoltLite read-path classification,
   - confirmed facts versus hypotheses,
   - focused regression target,
   - unanswered questions and follow-up work.

   Do not create implementation beads or launch follow-up formulas from the
   audit document step.

## Testing

This plan is document-only, so validation is artifact validation rather than
source-test execution.

- Verify the plan file exists under the default@ artifact root.
- Verify frontmatter declares `schema: gc.build.plan.v1`.
- Verify requirements coverage lists REQ-001 through REQ-007.
- Verify `jj status` shows only the plan and manifest changes expected for this
  document bead.
- For the later audit/fix workflow, run only focused tests such as:

  ```bash
  go test ./cmd/gc -run 'Test.*ScaleFromZero|Test.*NoScaleCheck|Test.*Named.*Demand'
  go test ./internal/beads -run 'TestCachingStoreCachedReadyDeclinesAfterDroppedRoutingEvent|Test.*DoltLite.*Ready|Test.*DoltLite.*Count'
  ```

Do not run the full test suite for this audit-planning task.

## Rollout

The rollout is the managed-document handoff:

1. Save `plan.md` in the default@ artifact root.
2. Update `manifest.json` with the plan path, schema, SHA-256 hash, and jj
   change id.
3. Record the same path, schema, hash, and change id on the plan bead.
4. Record the workflow root's latest `gc.docs.change_id`.
5. Close the plan bead with `gc.outcome=pass`.

No binary rebuild, service restart, source merge, push, or PR is part of this
rollout.

## Open Questions

- Which direct-discovery artifact should the audit use as the canonical proof
  that normal worker-routed work exists?
- Does the currently running controller use the local or origin variant of
  `pr/runtime-ready-demand-snapshot`?
- Is the missing route caused by demand snapshot reuse, desired-state merge
  logic, native DoltLite ready reads, or stale cached ready rows?
- Should the durable regression live in `cmd/gc` demand tests or
  `internal/beads` cache/native-read tests after evidence narrows the cause?
