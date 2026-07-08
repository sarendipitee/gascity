# Design Review: Session Model Unification

## Verdict

Approved after amendments.

## Scope

- Design: `engdocs/design/session-model-unification.md`
- Context reviewed:
  - `engdocs/architecture/dispatch.md`
  - `engdocs/design/session-reconciler-tracing.md`
  - `engdocs/architecture/beads.md`

## Amendments Applied

1. Reconciled the `gc.run_target` section with the dispatch architecture's
   temporary migration fallback. The design now says `gc.routed_to` is the
   normal persisted routing key while allowing the existing workflow-root-only
   `gc.run_target` fallback during the migration window.
2. Added a normative `Controller demand snapshot reuse` section. It defines
   when patrol demand snapshots may be reused, which inputs invalidate them,
   and the fail-open rule for partial or failed ready-work invalidation reads.

## Findings

No blocking findings remain after the amendments.

The reviewed design now preserves the dispatch contract that controller
demand and worker claim predicates stay symmetric, including the bounded
`gc.run_target` migration fallback. It also aligns with the reconciler tracing
design by making demand snapshot load/reuse an explicit decision point with
well-defined invalidation inputs instead of an implicit cache.

## Evidence

- Dispatch requires the controller count path and worker claim path to share
  the `gc.routed_to` predicate and the workflow-root-only `gc.run_target`
  migration predicate.
- Beads architecture establishes `gc.*` metadata as engine-owned vocabulary,
  so routing and migration keys need explicit ownership and canonical forms.
- Reconciler tracing treats the demand snapshot load as a concrete trace site;
  the design now gives that site a reviewable correctness contract.

## Residual Risk

The implementation should keep tests around the snapshot invalidation
boundary, especially newly ready routed work, assignment changes that remove
generic demand, partial store reads, and custom `scale_check` disabling the
event-backed snapshot path.
