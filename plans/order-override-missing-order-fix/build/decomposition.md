---
schema: gc.build.decomposition.v1
workflow:
  id: gc-4ycl
  formula: build-basic
methodology:
  pack: gascity
  name: build-basic
producer:
  formula: build-basic
  stage: decompose
  attempt: 1
status: approved
trace:
  upstream:
    - path: /data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/requirements.md
      hash: bead:gc-e6og
    - path: /data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/investigation-input.md
      hash: sha256:dc33d98230e774fbd04f9b85d1de6c92a24f0c9fd38b54b2fd8b95b754f80a7b
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
    - path: /data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/implementation-plan.md
      hash: sha256:a8c75ac93ee326d4e97002c9762d0f99991bfb160c35c63843b89414533e2499
    - path: /data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/plan-review-report.md
      hash: sha256:8459e8bd1e7548d1787d9c1b999e9bb34d36e55423cf6341e4d0d8f339e62cfd
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

# Decomposition: Fix Missing Order Override Warning

## Summary

This decomposition creates four implementation work items for the approved
order override plan. The recorded requirements artifact is missing, so
requirements traceability is grounded in the approved implementation plan and
the investigation input that carried the acceptance criteria.

| ID | Status |
| --- | --- |
| REQ-001 | covered |
| REQ-002 | covered |
| REQ-003 | covered |
| REQ-004 | covered |

## Selected Downstream Formulas

- Implementation formula: `implement`
- Work-item formula: `do-work-item`
- Code review formula: `review`
- Review-fix formula: `fix-loop-base`

## Implementation Convoy

Implementation convoy: `gc-0ih8`

Convoy name: `order-override-missing-order-fix-implementation`

The convoy contains four freshly-created work-item beads:

| Bead | Title | Trace | Dependencies |
| --- | --- | --- | --- |
| `gc-gja7` | Add order override tombstone unit tests | `REQ-001`, `REQ-002`, `REQ-004` | none |
| `gc-b4ui` | Add order scan missing-override contract tests | `REQ-001`, `REQ-002`, `REQ-004` | none |
| `gc-e1v1` | Implement disabled missing-order override tombstones | `REQ-001`, `REQ-002` | `gc-gja7`, `gc-b4ui` |
| `gc-67ln` | Document disabled optional-order override tombstones | `REQ-003` | none |

The implementation bead is intentionally ordered behind the two test beads so
the behavior is pinned before `internal/orders/override.go` changes.

## Work Items

### `gc-gja7`: Add order override tombstone unit tests

Scope:
Update `internal/orders/override_test.go` for disabled-only missing-order
tombstones and preserved invalid override diagnostics.

Acceptance criteria:

- Cover globally absent order names with `Enabled` explicitly set to false,
  including city-scope and rig-scoped overrides for `jjw-workspace-report`.
- Preserve diagnostics for `Enabled` nil or true, mutating missing overrides,
  and existing order names with the wrong rig scope.
- Focused proof: `go test ./internal/orders -run TestApplyOverrides`.

Trace:
`REQ-001`, `REQ-002`, `REQ-004`; implementation plan section
`Implementation Task Boundaries`.

### `gc-b4ui`: Add order scan missing-override contract tests

Scope:
Update `cmd/gc/order_scan_contract_test.go` and keep
`cmd/gc/order_dispatch_test.go` coverage for non-fatal but visible invalid
override errors.

Acceptance criteria:

- A minimal city with one unrelated valid order and a disabled missing override
  for `jjw-workspace-report` loads without a `not found` warning.
- The paired enabled or mutating missing override case still emits the
  `not found` diagnostic.
- Focused proof:
  `go test ./cmd/gc -run 'TestOrderScanContract(DisabledMissingOverrideIsSilent|EnabledMissingOverrideStillWarns|OverrideEnabledFalseMarksOrderDisabled)|TestBuildOrderDispatcherOverrideNotFoundNonFatal'`.

Trace:
`REQ-001`, `REQ-002`, `REQ-004`; implementation plan section
`Implementation Task Boundaries`.

### `gc-e1v1`: Implement disabled missing-order override tombstones

Scope:
Update `internal/orders/override.go` so only disabled-only overrides for
globally absent order names skip `notFoundError`.

Acceptance criteria:

- `ApplyOverrides` continues instead of returning `notFoundError` only for a
  disabled-only tombstone whose order name is globally absent.
- A small unexported helper, such as
  `isDisabledMissingOrderTombstone(ov Override, aa []Order) bool`, returns true
  only when `Enabled` is explicitly false, no other override fields are set,
  and no discovered order has the same `Name` regardless of rig.
- Enabled, nil-enabled, mutating, and wrong-rig overrides retain the existing
  diagnostics.
- Focused proof: both focused test commands from the test beads.

Trace:
`REQ-001`, `REQ-002`; implementation plan section
`Implementation Task Boundaries`.

### `gc-67ln`: Document disabled optional-order override tombstones

Scope:
Update `docs/tutorials/07-orders.md` and any generated config reference source
required by the schema or documentation pipeline.

Acceptance criteria:

- Explain near the existing `enabled = false` examples that disabled-only
  overrides may act as tombstones for optional orders that are not installed.
- Preserve the documented rule that enabled, mutating, or mis-scoped missing
  overrides remain configuration diagnostics.
- Respect generated-reference policy before editing
  `docs/reference/config.md` or `docs/reference/schema/city-schema.*`.
- Proof: review docs diff and generated-reference policy.

Trace:
`REQ-003`; implementation plan section `Implementation Task Boundaries`.
