---
schema: gc.build.decomposition.v1
workflow:
  id: gc-b7tg
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
    - path: plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/plan-review.md
      hash: sha256:6a02e1b4eb48f4c935745cf4df914f032839efda7c49b360a03d62ac20f13412
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

# Decomposition: Controller Demand vs Upstream Audit

## Summary

This decomposition preserves the approved document-only audit scope. The
implementation drain should produce a manifest-managed audit handoff that
compares controller demand behavior against the resolved upstream baseline,
classifies the relevant DoltLite and cache read paths, and identifies a focused
regression target for normal worker-routed demand.

No source code changes, cleanup, upstream reconciliation, PR work, or new source
workspaces are part of this decomposition.

| ID | Status |
| --- | --- |
| REQ-001 | covered |
| REQ-002 | covered |
| REQ-003 | covered |
| REQ-004 | covered |
| REQ-005 | covered |
| REQ-006 | covered |
| REQ-007 | covered |

## Selected Downstream Formulas

- Implementation formula: `jj-implement`
- Separate-session drain formula: `jj-do-work`
- Same-session item formula: `jj-do-work-item`
- Review formula: `jj-review`
- Review-fix formula: `jj-fix-loop`

The workflow root is configured with `drain_policy=separate`, so the
implementation stage should drain through `jj-do-work` while passing
`docs_workspace`, `docs_workspace_path`, `docs_artifact_root`, and
`manifest_path` from the manifest.

## Implementation Convoy

Implementation convoy: `gc-wbd5`

Convoy title: `input convoy for gc-o1ol`

Live work items:

| Bead | Title | Trace |
| --- | --- | --- |
| `gc-o1ol` | Audit controller demand against upstream | `REQ-001` through `REQ-007` |

This JJ-managed decomposition document does not create additional live beads.
The existing convoy remains the implementation boundary, and the work item
should write its audit result as a managed document under
`plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream`.

## Work Items

### `gc-o1ol`: Audit controller demand against upstream

Scope:
Produce the audit handoff described by the approved requirements and plan. Keep
the work read-only except for managed workflow documents.

Required audit slices:

1. Confirm the JJ document context.
   - Verify the current document change is described for this audit work.
   - Confirm the working copy has no unrelated source edits before writing the
     audit document.
   - Trace: `REQ-004`, `REQ-005`.

2. Resolve upstream and divergent runtime-demand baselines.
   - Record `jj log -r main --no-graph`,
     `jj log -r pr/runtime-ready-demand-snapshot --no-graph`, and
     `jj bookmark list` evidence.
   - Identify whether the running behavior matches the local or origin
     runtime-demand variant when the evidence supports a conclusion.
   - Trace: `REQ-002`, `REQ-007`.

3. Reproduce or restate the controller-demand symptom.
   - Capture the direct-discovery path that sees normal worker-routed work.
   - Capture the controller trace or existing trace artifact showing
     `scale_check_counts`.
   - Distinguish normal worker routes from `core.control-dispatcher` routes.
   - If the symptom no longer reproduces, state that explicitly and preserve
     previous evidence as historical context.
   - Trace: `REQ-001`.

4. Compare local demand code against the resolved upstream baseline.
   - Inspect `cmd/gc/build_desired_state.go` and `cmd/gc/city_runtime.go`.
   - Inspect focused demand tests under `cmd/gc`.
   - Separate confirmed facts from hypotheses for demand counting, snapshot
     reuse, fingerprinting, and desired-state merging.
   - Trace: `REQ-002`, `REQ-003`, `REQ-007`.

5. Classify the DoltLite and cache read paths.
   - Inspect `internal/beads/doltlite_read_store.go`,
     `internal/beads/bdstore.go`, `internal/beads/doltlite_count.go`,
     `internal/beads/doltlite_read_store_test.go`, and
     `internal/beads/caching_store_test.go`.
   - State whether controller demand uses native DoltLite reads, CLI bead
     reads, cached ready rows, live ready rows, or a combination.
   - Leave gaps as unanswered questions rather than filling them with
     inference.
   - Trace: `REQ-006`.

6. Identify focused regression coverage.
   - Prefer a narrow `cmd/gc` demand test when the missing behavior is
     pool/template or named-session demand.
   - Choose an `internal/beads` cache/native-read test only if the evidence
     proves stale or divergent ready rows are responsible.
   - Expected assertions should reference desired state or
     `scale_check_counts` containing the normal worker route that direct
     discovery sees.
   - Trace: `REQ-003`.

7. Write the audit handoff.
   - Include baseline revisions and commands used, symptom evidence,
     upstream/local comparison findings, DoltLite read-path classification,
     confirmed facts versus hypotheses, focused regression target, unanswered
     questions, and follow-up work.
   - Do not create implementation beads, launch follow-up formulas, change
     source code, or rewrite JJ history from the audit document step.
   - Trace: `REQ-004`, `REQ-005`.

Focused proof:
The audit handoff document records path, schema, hash, and JJ change ID in the
workflow manifest and on the workflow root bead.
