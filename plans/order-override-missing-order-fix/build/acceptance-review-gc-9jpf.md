---
schema: gc.build.acceptance-review.v1
review_bead: gc-9jpf
workflow_root: gc-4ycl
verdict: iterate
context_path: /data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/code-review-context.md
created_at: 2026-06-25T15:00:00Z
---

# Acceptance Review: Order Override Missing-Order Fix

## Verdict

`iterate`

The requested behavior is implemented correctly in the recorded implementation
source anchor: disabled-only overrides for globally absent optional orders are
accepted as tombstones, while enabled, nil-enabled, mutating, and wrong-rig
missing overrides still produce diagnostics.

Acceptance is blocked by task-boundary drift in the source anchors. The review
contract asks whether the factory avoided out-of-scope changes; two source
anchors changed files outside the decomposition scopes they were assigned.

## Required Findings

### 1. `gc-b4ui` implements core override behavior outside its scan-contract scope

Reference: `decomposition.md`, section `gc-b4ui: Add order scan missing-override contract tests`.

The decomposition scope for `gc-b4ui` says to update
`cmd/gc/order_scan_contract_test.go` and keep existing dispatcher coverage. The
recorded commit `d0e00a6964190dd5a5e54148de2754dd0db68acb` also changes:

- `/data/projects/doltlite-gascity/gascity/gc-urmg-prepare-item-worktree/worktrees/gc-b4ui/internal/orders/override.go`
- `/data/projects/doltlite-gascity/gascity/gc-urmg-prepare-item-worktree/worktrees/gc-b4ui/internal/orders/override_test.go`

Proof:

```bash
git show --name-status --format='%h %s' d0e00a6964190dd5a5e54148de2754dd0db68acb
```

Observed changed files:

```text
d0e00a6964 Handle missing disabled order overrides
M cmd/gc/order_scan_contract_test.go
M internal/orders/override.go
M internal/orders/override_test.go
```

The implementation proof is not just incidental formatting: `internal/orders/override.go`
adds `isMissingDisableNoop` at line 83 and wires it into `ApplyOverrides` at
line 67. That belongs to the `gc-e1v1` implementation task, not the scan
contract test task.

Smallest required fix: rework the `gc-b4ui` source anchor so it contains only
the scan-contract test changes required by its decomposition item, or revise
the decomposition/task summary to explicitly make `gc-b4ui` an implementation
anchor and regenerate review context accordingly.

### 2. `gc-e1v1` duplicates test-anchor changes outside its implementation-only scope

Reference: `decomposition.md`, section `gc-e1v1: Implement disabled missing-order override tombstones`.

The decomposition scope for `gc-e1v1` says to update
`internal/orders/override.go` so only disabled-only globally absent order names
skip `notFoundError`. The recorded commit
`1821d0aaa74c55551c8e48d477b6d62ffbc27ee9` also changes:

- `/data/projects/doltlite-gascity/gascity/gc-h6lo-generate-root-task-stage-report/worktrees/gc-e1v1/internal/orders/override_test.go`
- `/data/projects/doltlite-gascity/gascity/gc-h6lo-generate-root-task-stage-report/worktrees/gc-e1v1/cmd/gc/order_scan_contract_test.go`

Proof:

```bash
git show --name-status --format='%h %s' 1821d0aaa74c55551c8e48d477b6d62ffbc27ee9
```

Observed changed files:

```text
1821d0aaa7 Accept disabled missing order tombstones
M cmd/gc/order_scan_contract_test.go
M internal/orders/override.go
M internal/orders/override_test.go
```

Those test changes overlap the `gc-gja7` unit-test and `gc-b4ui`
scan-contract source-anchor responsibilities.

Smallest required fix: make `gc-e1v1` carry only the `internal/orders/override.go`
implementation change, with proof run against the test-anchor work, or revise
the decomposition/task summaries so the expanded file set is intentional and
traceable.

## Accepted Behavior Check

The core implementation in `gc-e1v1` matches the acceptance criteria:

- `ApplyOverrides` continues only when `Enabled` is explicitly false.
- The tombstone helper rejects any mutation fields: trigger, interval,
  schedule, check, on, pool, timeout, idempotent, or env.
- The helper rejects wrong-rig cases where any discovered order has the same
  name, preserving the existing not-found diagnostic.
- Documentation in `gc-67ln` explains disabled-only optional-order tombstones
  and preserves the diagnostic rule for enabled, mutating, or mis-scoped
  missing overrides.

## Verification Rerun

Focused checks rerun during this review:

```text
go test ./internal/orders -run TestApplyOverrides -count=1
PASS

go test ./cmd/gc -run 'TestOrderScanContract(DisabledMissingOverrideIsSilent|EnabledMissingOverrideStillWarns|OverrideEnabledFalseMarksOrderDisabled)|TestBuildOrderDispatcherOverrideNotFoundNonFatal' -count=1
PASS

go vet ./cmd/gc ./internal/orders
PASS

go test ./internal/docgen -run TestCitySchemaOrderOverrideIncludesLegacyGateAlias -count=1
PASS

go test ./test/docsync -run TestSchemaFreshness -count=1
PASS

go test ./test/docsync -run TestLocalMarkdownLinks -count=1
PASS
```

Known repo-wide blockers recorded in the review context were not treated as
implementation failures: `go vet ./...` fails on the pre-existing
`tmpinspect/main.go` issue, and `make check-docs` fails on pre-existing
docsync directory coverage for `gc-plans` and `tools`.
