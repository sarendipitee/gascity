# Starter Review Synthesis: Order Override Missing-Order Fix

Recovered by mayor from bead metadata and durable review artifacts after the
original synthesis artifact was missing from the build artifact root.

## Verdict

`iterate`

The starter review did not fully approve. The fix lane should apply the
smallest focused cleanup in the recorded source-anchor worktrees, then write a
review-fix summary and set `code_review.verdict` according to the post-fix
state.

## Source Artifacts

| Lane | Bead | Verdict | Durable artifact |
| --- | --- | --- | --- |
| Acceptance and correctness | `gc-9jpf` | `iterate` | `acceptance-review-gc-9jpf.md` |
| Test evidence | `gc-bbr0` | `approve` | Metadata recorded `test-evidence-review.md`, but the file is missing. |
| Simplicity and maintainability | `gc-k5i2` | `iterate` | Metadata recorded `simplicity-review.md`, but the file is missing. |

Review the source-anchor worktrees recorded in `code-review-context.md`, not
the launcher rig root. An unchanged launcher checkout is not itself a required
fix before publish.

## Required Fixes

### 1. Keep core override behavior in the correct source-anchor boundary

Source lane: acceptance and correctness (`gc-9jpf`).

The acceptance lane found that source anchor `gc-b4ui` implemented core override
behavior outside its scan-contract scope. The reported files are:

- `/data/projects/doltlite-gascity/gascity/gc-urmg-prepare-item-worktree/worktrees/gc-b4ui/internal/orders/override.go`
- `/data/projects/doltlite-gascity/gascity/gc-urmg-prepare-item-worktree/worktrees/gc-b4ui/internal/orders/override_test.go`

Smallest useful fix: keep scan-contract coverage in the scan-contract anchor,
but move or remove production override behavior and shared unit-test ownership
from that anchor so the implementation boundary matches the decomposition.

### 2. Remove duplicated test-anchor changes from the implementation-only anchor

Source lane: acceptance and correctness (`gc-9jpf`).

The acceptance lane found that source anchor `gc-e1v1` duplicated test-anchor
changes outside its implementation-only scope. The reported files are:

- `/data/projects/doltlite-gascity/gascity/gc-h6lo-generate-root-task-stage-report/worktrees/gc-e1v1/internal/orders/override_test.go`
- `/data/projects/doltlite-gascity/gascity/gc-h6lo-generate-root-task-stage-report/worktrees/gc-e1v1/cmd/gc/order_scan_contract_test.go`

Smallest useful fix: keep the implementation anchor focused on the production
change and leave task-specific tests in their owning test anchors, unless the
decomposition is updated to make the broader ownership explicit.

### 3. Remove unrelated generated docs/schema refresh noise

Source lane: simplicity and maintainability (`gc-k5i2`).

The simplicity lane closed with iteration because generated config docs/schema
include unrelated refresh noise. The review context lists the documentation
anchor `gc-67ln` as changing:

- `docs/tutorials/07-orders.md`
- `internal/config/config.go`
- `docs/reference/config.md`
- `docs/reference/schema/city-schema.json`
- `docs/reference/schema/city-schema.txt`

Smallest useful fix: keep documentation directly needed for disabled-only
optional-order tombstones and remove unrelated generated config/schema churn.

## Missing Evidence

The test evidence lane approved in bead metadata:

- `gc-bbr0` set `code_review.test_evidence_verdict=approve`
- `gc-bbr0` closed with reason `Build-basic test evidence review approved.`

The lane's metadata points to `test-evidence-review.md`, but that file is not
present in the artifact root. Do not treat that missing lane file as a product
defect; record the artifact-loss condition in the review-fix summary.

## Residual Risks

- The generated requirements artifact is missing for this workflow; the review
  context says to use the investigation input plus approved plan/decomposition
  as the requirements basis.
- `go vet ./...` and pre-commit were reported as blocked by the pre-existing
  tracked `tmpinspect/main.go`; use focused proof commands for this fix unless
  that unrelated issue is intentionally in scope.
- This synthesis is reconstructed, not the original report emitted by
  `gc-48gn`. Preserve this file now that downstream metadata points to it.

## Fix-Lane Close Guidance

After applying the focused fixes, write the review-fix summary under the build
artifact root and close `gc-9psc` with:

- `gc.outcome=pass`
- `code_review.verdict=done` if acceptance, test evidence, and simplicity are
  all satisfied after the pass
- `code_review.verdict=iterate` if required fixes remain
- `code_review.report_path=<review-fix summary path>`
- `code_review.output_path=<review-fix summary path>`
