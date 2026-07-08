---
schema: gc.build.implementation-summary.v1
workflow:
  id: gc-4ycl
  formula: build-basic
methodology:
  pack: gascity
  name: build-basic
producer:
  formula: build-basic
  stage: summarize-implementation
  attempt: 1
status: approved
trace:
  upstream:
    - path: beads/gc-0ih8
      hash: bead:gc-0ih8
    - path: beads/gc-gja7
      hash: bead:gc-gja7
    - path: beads/gc-67ln
      hash: bead:gc-67ln
    - path: beads/gc-b4ui
      hash: bead:gc-b4ui
    - path: beads/gc-e1v1
      hash: bead:gc-e1v1
    - path: /data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-plan.md
      hash: sha256:a8c75ac93ee326d4e97002c9762d0f99991bfb160c35c63843b89414533e2499
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
    - path: /data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/decomposition.md
      hash: sha256:da1521571dd1907b7a543a8616e35fe0be291c93d7e0dcad395912188a725935
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
    - path: /data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-summary-gc-2j2y.md
      hash: sha256:7003047024e3697d41a98c63f9c2d4e19e7abc88305084e3c379d42efedcd150
      ids:
        - REQ-001
        - REQ-002
        - REQ-004
    - path: /data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-summary-gc-anyc.md
      hash: sha256:be91fc3e4838d697aa29c67a5342fd9875352f6657c3cd5cc0beba57a8d954ff
      ids:
        - REQ-003
    - path: /data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/gc-hyls-implementation-summary.md
      hash: sha256:3262d3457b7e7bb647a16bfec6812baf1392e0c1c980a1490c588c5a28f7844f
      ids:
        - REQ-001
        - REQ-002
        - REQ-004
    - path: /data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-summary-gc-7wwo.md
      hash: sha256:2ededd573658c7c2fce48a10ceb0c1cc5efe9c7a25ef62a6bf6216aabbdfd9b4
      ids:
        - REQ-001
        - REQ-002
  coverage:
    - id: REQ-001
      status: covered
    - id: REQ-002
      status: covered
    - id: REQ-003
      status: covered
    - id: REQ-004
      status: covered
---
# Implementation Summary: Order Override Missing-Order Fix

## Summary

The build implementation convoy `gc-0ih8` completed four source anchors:
`gc-gja7`, `gc-67ln`, `gc-b4ui`, and `gc-e1v1`. Their child implementation
workflows all closed with `gc.outcome=pass`: `gc-a4i7`, `gc-dj3m`, `gc-8vfw`,
and `gc-rzkh`.

The accepted behavior is implemented and documented: disabled-only overrides
for globally absent optional order names are treated as tombstones, while
enabled, nil-enabled, mutating, and wrong-scope overrides still report the
existing missing-order diagnostics.

| ID | Status |
| --- | --- |
| REQ-001 | covered |
| REQ-002 | covered |
| REQ-003 | covered |
| REQ-004 | covered |

Upstream item summaries:

- `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-summary-gc-2j2y.md`
- `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-summary-gc-anyc.md`
- `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/gc-hyls-implementation-summary.md`
- `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-summary-gc-7wwo.md`

## Intended Behavior

Users may keep `[[orders.overrides]]` entries that contain an order `name` and
`enabled = false` as optional-order tombstones. If no discovered order has that
name at any scope, the override is accepted silently and no repeated patrol or
scanner warning is emitted.

Missing-order overrides remain diagnostics when they would enable an order,
leave `enabled` unset, mutate interval/env/check behavior, or target the wrong
rig scope for an order name that exists elsewhere. Existing dispatcher coverage
for non-fatal invalid override diagnostics remains in place.

## Changed Files

- `internal/orders/override.go`: added the disabled-only missing-order tombstone
  path and preserves `not found` errors for active, mutating, nil-enabled, and
  wrong-scope overrides.
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

Relevant source-item commits recorded by the implementation summaries:
`7fd4cf1e142b`, `9d266391df`,
`d0e00a6964190dd5a5e54148de2754dd0db68acb`, and `1821d0aaa74c`.

## Verification

First verification commands:

- `go test ./internal/orders -run TestApplyOverrides -count=1` initially failed
  in `gc-gja7` because the new tombstone tests preceded the implementation.
- `go test ./cmd/gc -run 'TestOrderScanContractMissing(DisabledOverrideIsSilent|ActiveOrMutatingOverrideWarns)$'`
  timed out during package build in `gc-b4ui` before reaching a useful
  assertion.

Final proof commands and observed results:

- `go test ./internal/orders -run TestApplyOverrides -count=1`: pass.
- `go test ./cmd/gc -run 'Test(OrderScanContract(MissingDisabledOverrideIsSilent|MissingActiveOrMutatingOverrideWarns|OverrideEnabledFalseMarksOrderDisabled)|BuildOrderDispatcherOverride(NotFoundNonFatal|DisablesDropsFromDispatcher))$'`: pass.
- `go test ./cmd/gc -run 'TestOrderScanContract(DisabledMissingOverrideIsSilent|EnabledMissingOverrideStillWarns|OverrideEnabledFalseMarksOrderDisabled)|TestBuildOrderDispatcherOverrideNotFoundNonFatal' -count=1`: pass.
- `go vet ./cmd/gc ./internal/orders`: pass.
- `go run ./cmd/genschema`: pass.
- `go test ./internal/docgen -run TestCitySchemaOrderOverrideIncludesLegacyGateAlias -count=1`: pass.
- `go test ./test/docsync -run TestSchemaFreshness -count=1`: pass.
- `go test ./test/docsync -run TestLocalMarkdownLinks -count=1`: pass.
- `git diff --check`: pass.
- `go vet ./...`: fail because pre-existing tracked `tmpinspect/main.go`
  references undefined `config.LoadCityConfig`.
- `make check-docs`: fail because of pre-existing docsync directory coverage
  issues for `gc-plans` and `tools`; no changed-doc link failure was reported.
- `.githooks/pre-commit`: active and attempted by source-item workers, but did
  not complete cleanly because of the unrelated repo-wide vet blocker or a
  stalled `golangci-lint --fix` invocation.
- Item-level build artifact validation passed for the recorded per-item
  summaries using available validator script copies.

## Remaining Risks

The root requirements artifact path recorded by the workflow does not exist,
and the root review report path is also absent. Requirement coverage is
therefore grounded in the approved implementation plan and decomposition, both
of which record `REQ-001` through `REQ-004` as covered.

Repository-wide gates still have unrelated pre-existing blockers:
`tmpinspect/main.go` prevents `go vet ./...`, and docsync directory coverage
prevents `make check-docs`. The launcher worktree for this root step has
`.gc/` but no `.gc/scripts/checks/build-artifact-valid.sh`, so local validation
may require the pack validator script path used by sibling summaries.
