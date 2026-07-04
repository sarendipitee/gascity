# Starter Review Fix Summary: Order Override Missing-Order Fix

## Outcome

Verdict: iterate.

No implementation source changes were made in this pass. The preserved starter
review synthesis reports required fixes and missing evidence, and the workflow
handoff itself is incomplete: the synthesis artifact is present in jj change
`zrxomlvkwruu` but is absent from `default@` and the current workspace path
recorded in bead metadata.

## Evidence Checked

- Claimed bead: `gc-9psc`
- Current step: `review.apply-review-findings`
- Current step ref:
  `review.build-basic-review-loop.iteration.1.review.apply-review-findings`
- Workflow root: `gc-4ycl`
- Recorded synthesis path:
  `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/starter-review-synthesis.md`
- `default@` check: `starter-review-synthesis.md` is not present under
  `../plans/order-override-missing-order-fix/build/`.
- Preserved docs change: `zrxomlvkwruu` (`document: preserve starter review synthesis`)
  adds `../plans/order-override-missing-order-fix/build/starter-review-synthesis.md`.
- Mayor notification: sent mail `dg-wisp-730f338094` with the workflow handoff
  failure details.

## Remaining Required Fixes

The preserved synthesis verdict is `iterate` and lists these required fixes:

1. Reconcile `gc-b4ui` source-anchor scope drift. The decomposition scoped the
   anchor to `cmd/gc/order_scan_contract_test.go`, but the recorded commit also
   changed `internal/orders/override.go` and `internal/orders/override_test.go`.
2. Reconcile `gc-e1v1` source-anchor scope drift. The decomposition scoped the
   anchor to `internal/orders/override.go`, but the recorded commit also changed
   `cmd/gc/order_scan_contract_test.go` and `internal/orders/override_test.go`.
3. Address simplicity-review generated-artifact noise. The simplicity lane
   reported unrelated generated config docs/schema refresh noise.

## Missing Evidence

- `test-evidence-review.md` is declared as the test-evidence lane output path,
  but the file is absent from the materialized artifact set.
- `simplicity-review.md` is declared as the simplicity lane output path, but the
  file is absent from the materialized artifact set.

## Workflow Issue To Fix

Downstream review-fix workers need a durable way to open the synthesis artifact.
Either the synthesize step should land/preserve `starter-review-synthesis.md` in
`default@` before routing `review.apply-review-findings`, or the workflow should
record and pass the jj docs change/revset containing generated review artifacts.
Without that handoff, this step can falsely report the synthesis as missing even
though it exists in a separate jj change.
