---
schema: gc.build.code-review-context.v1
workflow:
  id: gc-4ycl
  formula: build-basic
methodology:
  pack: gascity
  name: build-basic
producer:
  bead: gc-uqcd
  stage: review-setup
status: prepared
---

# Code Review Context: Order Override Missing-Order Fix

## Review Scope

Review the source-anchor worktrees listed in this file, not the launcher rig
checkout. The launcher rig root may remain unchanged until an explicit publish
step. The implementation source of truth is the closed source anchors and their
recorded worktrees, commits, summaries, and proof commands.

This starter factory uses three review lanes: acceptance and correctness, test
evidence, and simplicity and maintainability.

## Artifact Index

| Artifact | Path | Status |
| --- | --- | --- |
| Requirements | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/requirements.md` | Missing. Requirements bead `gc-e6og` closed with `gc.outcome=fail` / `gc.failure_class=hard` / `gc.failure_reason=control_dispatch_error`. |
| Investigation input | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/investigation-input.md` | Present. Used as the requirements source by the approved plan and decomposition. |
| Implementation plan | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-plan.md` | Present and approved. |
| Plan review report | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/plan-review-report.md` | Present. |
| Decomposition | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/decomposition.md` | Present and approved. |
| Canonical implementation summary | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-summary.md` | Present. |
| Per-item summary: unit tests | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-summary-gc-2j2y.md` | Present. |
| Per-item summary: docs | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-summary-gc-anyc.md` | Present. |
| Per-item summary: scan contract tests | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/gc-hyls-implementation-summary.md` | Present. |
| Per-item summary: implementation | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-summary-gc-7wwo.md` | Present. |

## Requirements Basis

The generated requirements artifact is missing. The approved implementation
plan and decomposition explicitly record that gap and trace the work to the
investigation input acceptance criteria:

| ID | Requirement Basis | Status |
| --- | --- | --- |
| REQ-001 | The repeated `jjw-workspace-report` missing-order warning no longer appears for disabled override entries in this city. | Covered |
| REQ-002 | Enabled missing-order override diagnostics remain covered by tests. | Covered |
| REQ-003 | The chosen behavior is documented or already matches existing documentation. | Covered |
| REQ-004 | Relevant tests are added or updated around disabled order overrides. | Covered |

## Approved Implementation Plan Summary

Change order override application so a disabled-only override for an order name
that is not installed is treated as a valid optional-order tombstone, not as a
patrol/scanner warning. Preserve diagnostics for enabled missing-order
overrides, nil-enabled missing-order overrides, mutating missing-order
overrides, and wrong-rig overrides against an existing order name.

Primary code surfaces named by the plan:

- `internal/orders/override.go`: central `ApplyOverrides` behavior.
- `internal/orderdiscovery/discovery.go`: order discovery calls override
  application.
- `cmd/gc/cmd_order.go`: command path emits override warnings.
- `internal/orders/override_test.go`: focused override unit tests.
- `cmd/gc/order_scan_contract_test.go`: scanner contract coverage.
- `docs/tutorials/07-orders.md`, `internal/config/config.go`, and generated
  config/schema docs: documentation and generated reference updates.

## Decomposition Summary

Implementation convoy: `gc-0ih8`

| Source Anchor | Title | Requirements | Dependencies |
| --- | --- | --- | --- |
| `gc-gja7` | Add order override tombstone unit tests | REQ-001, REQ-002, REQ-004 | None |
| `gc-b4ui` | Add order scan missing-override contract tests | REQ-001, REQ-002, REQ-004 | None |
| `gc-e1v1` | Implement disabled missing-order override tombstones | REQ-001, REQ-002 | `gc-gja7`, `gc-b4ui` |
| `gc-67ln` | Document disabled optional-order override tombstones | REQ-003 | None |

## Source Anchors And Worktrees

| Source Anchor | Outcome | Commit | Worktree | Summary |
| --- | --- | --- | --- | --- |
| `gc-gja7` | pass | `7fd4cf1e142b596c56b8aff67164885bdfa3751c` | `/data/projects/doltlite-gascity/gascity/gc-l7qm-prepare-item-worktree/worktrees/gc-gja7` | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-summary-gc-2j2y.md` |
| `gc-67ln` | pass | `9d266391df` | `/data/projects/doltlite-gascity/gascity/gc-u5sn-prepare-item-worktree/worktrees/gc-67ln` | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-summary-gc-anyc.md` |
| `gc-b4ui` | pass | `d0e00a6964190dd5a5e54148de2754dd0db68acb` | `/data/projects/doltlite-gascity/gascity/gc-urmg-prepare-item-worktree/worktrees/gc-b4ui` | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/gc-hyls-implementation-summary.md` |
| `gc-e1v1` | pass | `1821d0aaa74c55551c8e48d477b6d62ffbc27ee9` | `/data/projects/doltlite-gascity/gascity/gc-h6lo-generate-root-task-stage-report/worktrees/gc-e1v1` | `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-summary-gc-7wwo.md` |

Note: `gc-67ln` records its commit and summary path in the close reason rather
than in `gc.implementation.*` metadata. Its `work_dir` is present in metadata.

## Changed File Summaries

- `internal/orders/override.go`: added the disabled-only missing-order
  tombstone path. The behavior still reports `not found` errors for active,
  mutating, nil-enabled, and wrong-scope overrides.
- `internal/orders/override_test.go`: added table coverage for city-scope and
  rig-scoped optional-order tombstones and negative cases for non-tombstone
  missing overrides.
- `cmd/gc/order_scan_contract_test.go`: added scanner contract coverage proving
  disabled missing overrides are silent and enabled or mutating missing
  overrides remain visible.
- `docs/tutorials/07-orders.md`: documented disabled-only optional-order
  tombstones and the diagnostic cases that remain invalid.
- `internal/config/config.go`: updated generated config-reference source
  comments for `OrdersConfig.Overrides` and `OrderOverride.Enabled`.
- `docs/reference/config.md`, `docs/reference/schema/city-schema.json`, and
  `docs/reference/schema/city-schema.txt`: regenerated from the updated config
  comments.

## Task Evidence

### `gc-gja7`: Unit Tests

Changed file:

- `internal/orders/override_test.go`

Evidence:

- Added behavior-pinning `TestApplyOverrides` cases for city-scope and
  rig-scoped disabled missing-order tombstones.
- Added negative cases for nil-enabled, enabled-true, mutating missing
  overrides, and disabled wrong-rig overrides.
- Initial and final focused test runs failed before implementation, as expected
  for a tests-first source anchor.
- Pre-commit ran and blocked on the pre-existing tracked `tmpinspect/main.go`
  `go vet ./...` failure, so the focused commit was created with
  `--no-verify` after the unrelated blocker was identified.

Proof commands recorded:

```bash
go test ./internal/orders -run TestApplyOverrides -count=1
git commit -m "Add order override tombstone unit tests"
git commit --no-verify -m "Add order override tombstone unit tests"
```

### `gc-b4ui`: Scan Contract Tests

Changed files:

- `internal/orders/override.go`
- `internal/orders/override_test.go`
- `cmd/gc/order_scan_contract_test.go`

Evidence:

- Added scan contract coverage for silent disabled missing overrides and visible
  active or mutating missing overrides.
- Preserved coverage for `enabled=false` on installed orders and dispatcher
  non-fatal warning behavior.
- First command timed out during package build before reaching a useful
  assertion.
- Final focused tests, focused vet, and artifact validation passed.
- Pre-commit ran and failed at the unrelated tracked `tmpinspect/main.go`
  `go vet ./...` blocker; `golangci-lint --fix` reported `0 issues` for
  `./cmd/gc ./internal/orders`.

Proof commands recorded:

```bash
go test ./cmd/gc -run 'TestOrderScanContractMissing(DisabledOverrideIsSilent|ActiveOrMutatingOverrideWarns)$'
go test ./internal/orders -run 'TestApplyOverrides' && go test ./cmd/gc -run 'Test(OrderScanContract(MissingDisabledOverrideIsSilent|MissingActiveOrMutatingOverrideWarns|OverrideEnabledFalseMarksOrderDisabled)|BuildOrderDispatcherOverride(NotFoundNonFatal|DisablesDropsFromDispatcher))$'
go vet ./cmd/gc ./internal/orders
GC_BEAD_ID=gc-hyls /data/projects/doltlite-gascity/gascity/gc-a4q1-write-canonical-implementation-summary/.gc/scripts/checks/build-artifact-valid.sh
```

### `gc-e1v1`: Implementation

Changed files:

- `internal/orders/override.go`
- `internal/orders/override_test.go`
- `cmd/gc/order_scan_contract_test.go`

Evidence:

- Implemented disabled-only missing-order tombstones for order overrides.
- Added `isDisabledMissingOrderTombstone` and skipped `notFoundError` only when
  the override is disabled-only and the order name is globally absent.
- Preserved diagnostics for enabled, mutating, nil-enabled, and wrong-rig
  missing overrides.
- Focused internal/orders and cmd/gc proof commands passed.
- `go vet ./...` failed on the pre-existing tracked `tmpinspect/main.go`
  undefined `config.LoadCityConfig` blocker.
- `.githooks/pre-commit` was active and ran for the staged change, but both
  attempts stalled inside `golangci-lint run --new-from-rev=HEAD --whole-files
  --fix ./cmd/gc ./internal/orders`; the runaway hook processes were
  terminated.
- Artifact validation passed.

Proof commands recorded:

```bash
go test ./internal/orders -run TestApplyOverrides -count=1
go test ./cmd/gc -run 'TestOrderScanContract(DisabledMissingOverrideIsSilent|EnabledMissingOverrideStillWarns|OverrideEnabledFalseMarksOrderDisabled)|TestBuildOrderDispatcherOverrideNotFoundNonFatal' -count=1
go vet ./...
GC_BEAD_ID=gc-7wwo /data/projects/doltlite-gascity/gascity/gc-a4q1-write-canonical-implementation-summary/.gc/scripts/checks/build-artifact-valid.sh
```

### `gc-67ln`: Documentation

Changed files:

- `docs/tutorials/07-orders.md`
- `internal/config/config.go`
- `docs/reference/config.md`
- `docs/reference/schema/city-schema.json`
- `docs/reference/schema/city-schema.txt`

Evidence:

- Documented disabled-only missing order overrides as optional-order
  tombstones in the order tutorial and generated config reference.
- Regenerated config and schema docs from the source comments.
- Schema, docgen, markdown-link, and whitespace checks passed where focused.
- `make check-docs` failed on pre-existing docsync directory coverage issues
  for `gc-plans` and `tools`; no changed-doc link failure was reported.
- `go vet ./...` failed on the pre-existing tracked `tmpinspect/main.go`
  undefined `config.LoadCityConfig` blocker.
- Pre-commit ran and blocked on the same unrelated vet error, so the focused
  commit was created with `--no-verify` after the unrelated blocker was
  identified.
- Artifact validation passed with the pack validator script.

Proof commands recorded:

```bash
go run ./cmd/genschema
make check-docs
go test ./internal/docgen -run TestCitySchemaOrderOverrideIncludesLegacyGateAlias -count=1
go test ./test/docsync -run TestSchemaFreshness -count=1
git diff --check
go test ./test/docsync -run TestLocalMarkdownLinks -count=1
go vet ./...
git commit -m "Document optional order override tombstones"
git commit --no-verify -m "Document optional order override tombstones"
GC_BEAD_ID=gc-anyc /data/projects/doltlite-gascity/gascity-packs/gascity/assets/scripts/checks/build-artifact-valid.sh
```

## Canonical Implementation Summary

The build implementation convoy `gc-0ih8` completed four source anchors:
`gc-gja7`, `gc-67ln`, `gc-b4ui`, and `gc-e1v1`. Their child implementation
workflows closed with `gc.outcome=pass`: `gc-a4i7`, `gc-dj3m`, `gc-8vfw`, and
`gc-rzkh`.

Accepted behavior:

- Disabled-only overrides for globally absent optional order names are
  tombstones and do not emit the repeated missing-order warning.
- Enabled, nil-enabled, mutating, and wrong-scope overrides still report the
  existing missing-order diagnostics.

Final proof commands and results recorded in the canonical summary:

| Command | Result |
| --- | --- |
| `go test ./internal/orders -run TestApplyOverrides -count=1` | pass |
| `go test ./cmd/gc -run 'Test(OrderScanContract(MissingDisabledOverrideIsSilent|MissingActiveOrMutatingOverrideWarns|OverrideEnabledFalseMarksOrderDisabled)|BuildOrderDispatcherOverride(NotFoundNonFatal|DisablesDropsFromDispatcher))$'` | pass |
| `go test ./cmd/gc -run 'TestOrderScanContract(DisabledMissingOverrideIsSilent|EnabledMissingOverrideStillWarns|OverrideEnabledFalseMarksOrderDisabled)|TestBuildOrderDispatcherOverrideNotFoundNonFatal' -count=1` | pass |
| `go vet ./cmd/gc ./internal/orders` | pass |
| `go run ./cmd/genschema` | pass |
| `go test ./internal/docgen -run TestCitySchemaOrderOverrideIncludesLegacyGateAlias -count=1` | pass |
| `go test ./test/docsync -run TestSchemaFreshness -count=1` | pass |
| `go test ./test/docsync -run TestLocalMarkdownLinks -count=1` | pass |
| `git diff --check` | pass |
| `go vet ./...` | fail: pre-existing tracked `tmpinspect/main.go` references undefined `config.LoadCityConfig` |
| `make check-docs` | fail: pre-existing docsync directory coverage issues for `gc-plans` and `tools`; no changed-doc link failure reported |
| `.githooks/pre-commit` | active and attempted; did not complete cleanly because of the unrelated repo-wide vet blocker or a stalled `golangci-lint --fix` invocation |

## Review Notes

- Do not treat the missing generated requirements artifact as an implementation
  source failure; the approved plan and decomposition already trace to the
  investigation input and record `gc-e6og` as failed.
- Do not treat an unchanged launcher checkout as a review failure. Review the
  source-anchor worktrees and commits above.
- Required review attention: behavior gate for disabled-only tombstones,
  preservation of invalid override diagnostics, focused test coverage, and
  generated docs/schema consistency.
- Known unrelated blockers recorded by source anchors: repo-wide `go vet ./...`
  fails on tracked `tmpinspect/main.go`, and `make check-docs` fails on
  pre-existing docsync directory coverage for `gc-plans` and `tools`.
