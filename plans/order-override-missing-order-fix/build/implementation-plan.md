---
schema: gc.build.plan.v1
workflow:
  id: gc-4ycl
  formula: build-basic
methodology:
  pack: gascity
  name: build-basic
producer:
  formula: build-basic
  stage: plan
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

# Implementation Plan: Fix Missing Order Override Warning

## Coverage

| ID | Status |
| --- | --- |
| REQ-001 | covered |
| REQ-002 | covered |
| REQ-003 | covered |
| REQ-004 | covered |

## Summary

Change order override application so a disabled-only override for an order name
that is not installed is treated as a valid tombstone, not a patrol/scanner
warning. This removes repeated diagnostics like
`orders.overrides[4]: order "jjw-workspace-report" (rig "gascity") not found`
for disabled overrides in `/data/projects/doltlite-gascity/city.toml`.

Preserve the current safety behavior for enabled or mutating overrides:
enabled missing-order overrides, interval/env/check edits for missing orders,
and rig-scope mismatches against an existing order name must still report a
clear diagnostic.

The generated requirements artifact recorded on the workflow root was not
available while authoring this plan:
`/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/requirements.md`.
The previous requirements step bead `gc-e6og` closed with `gc.outcome=fail`.
This plan is therefore grounded in the available investigation input artifact
and records the missing requirements artifact in trace metadata.

Requirement coverage:

- `REQ-001`: disabled missing order overrides no longer produce repeated patrol
  warnings.
- `REQ-002`: enabled missing order override diagnostics remain visible.
- `REQ-003`: the selected tombstone behavior is documented in order override
  docs.
- `REQ-004`: focused tests cover disabled missing override behavior and
  preserved enabled diagnostics.

## Current System

Order override behavior is centralized in
`internal/orders/override.go`. `ApplyOverrides` loops over configured
overrides, matches each by `Name` and `Rig`, applies field mutations to every
matching order, and returns `notFoundError` when no order matches. The current
logic does not distinguish `enabled = false` tombstones from active mutations;
a missing disabled override currently returns the same error as a missing
enabled override.

Shared order discovery lives in `internal/orderdiscovery/discovery.go`.
`ScanAll` scans city and rig order roots, builds the combined order slice, then
calls `orders.ApplyOverrides(allOrders, overridesFromConfig(...))`. If
`ApplyOverrides` returns an error, callers with `OnOverrideError` can keep the
scan non-fatal, but the command path still emits the warning through
`cmd/gc/cmd_order.go` `orderScanOptions`.

Existing tests already cover much of this path:

- `internal/orders/override_test.go` expects missing disabled overrides to
  error today, including cases for a plain missing name and wildcard missing
  name.
- `cmd/gc/order_scan_contract_test.go` pins post-override discovery behavior,
  including `enabled = false` on an installed order and rig-scoped override
  targeting.
- `cmd/gc/order_dispatch_test.go` verifies missing override errors are
  non-fatal for the dispatcher, but still visible in stderr.

Documentation for order overrides appears in `docs/tutorials/07-orders.md` and
generated reference material under `docs/reference/config.md` and
`docs/reference/schema/city-schema.*`. The tutorial already shows
`enabled = false` overrides; it does not explicitly state whether a
disabled-only override may reference an optional order that is not currently
installed.

## Proposed Implementation

Start with tests and make the behavior explicit at the override layer.

1. Update `internal/orders/override_test.go` so disabled-only tombstones for a
   globally absent order name succeed. Cover at least city-scope and rig-scoped
   inputs such as:
   `Override{Name: "jjw-workspace-report", Enabled: boolPtr(false)}` and
   `Override{Name: "jjw-workspace-report", Rig: "gascity", Enabled: boolPtr(false)}`.

2. Preserve diagnostic tests for real invalid overrides. Add or retain cases
   where a missing override still errors when:
   - `Enabled` is nil or true.
   - The override mutates any other field such as `Interval`, `Trigger`,
     `Schedule`, `Check`, `On`, `Pool`, `Timeout`, `Idempotent`, or `Env`.
   - The order name exists, but the supplied rig scope does not match. This
     keeps the existing helpful rig suggestion behavior for mis-scoped overrides.

3. Add command-level coverage in `cmd/gc/order_scan_contract_test.go`.
   Create a minimal city with one unrelated valid order, configure a disabled
   missing override for `jjw-workspace-report`, call `loadAllOrders`, and assert
   the call succeeds without writing a `not found` warning to stderr. Add the
   paired enabled or mutating missing override case and assert the warning is
   still present.

4. Implement the behavior in `internal/orders/override.go`. In
   `ApplyOverrides`, when no matching order is found, continue instead of
   returning `notFoundError` only when the override is a disabled-only tombstone
   for an order name with zero discovered instances. A small unexported helper
   keeps the rule visible:
   `isDisabledMissingOrderTombstone(ov Override, aa []Order) bool`.

   The helper should return true only when:
   - `ov.Enabled != nil && !*ov.Enabled`.
   - No other override fields are set.
   - No discovered order has `Name == ov.Name`, regardless of rig.

   That last condition is important. A disabled override for an existing name
   but wrong rig is probably a scope mistake, not an absent optional order, and
   should keep the current diagnostic.

5. Update comments near `ApplyOverrides` to state that disabled-only overrides
   may act as tombstones for optional orders that are not installed. Keep the
   rest of the matching contract intact.

6. Update `docs/tutorials/07-orders.md` to explain the tombstone case near the
   existing `enabled = false` override examples. If `docs/reference/config.md`
   and `docs/reference/schema/city-schema.*` are generated from Go comments or
   schema generation, update the source comment and regenerate rather than
   hand-editing generated output.

Implementation is deliberately limited to the override layer. No patrol-loop
special case is needed because all order consumers already flow through
`orderdiscovery.ScanAll` and `orders.ApplyOverrides`.

## Implementation Task Boundaries

Decompose this plan into implementation beads along these boundaries:

| Task | Scope | Acceptance Criteria | Focused Proof |
| --- | --- | --- | --- |
| Override unit tests | Update `internal/orders/override_test.go` for disabled-only missing-order tombstones and preserved invalid override diagnostics. | `REQ-001`, `REQ-002`, `REQ-004` | `go test ./internal/orders -run TestApplyOverrides` |
| Command contract tests | Update `cmd/gc/order_scan_contract_test.go` and keep `cmd/gc/order_dispatch_test.go` coverage for non-fatal but visible invalid override errors. | `REQ-001`, `REQ-002`, `REQ-004` | `go test ./cmd/gc -run 'TestOrderScanContract(DisabledMissingOverrideIsSilent|EnabledMissingOverrideStillWarns|OverrideEnabledFalseMarksOrderDisabled)|TestBuildOrderDispatcherOverrideNotFoundNonFatal'` |
| Override implementation | Update `internal/orders/override.go` so only disabled-only overrides for globally absent order names skip `notFoundError`. | `REQ-001`, `REQ-002` | Both focused test commands above |
| Documentation | Update `docs/tutorials/07-orders.md` and any generated config reference source required by the schema/doc pipeline. | `REQ-003` | Review docs diff and generated-reference policy before hand-editing generated files |

Each task is independently reviewable. Keep the override implementation bead
behind the test beads if the decomposition supports ordering; the desired rule
is small enough that implementation before tests would hide the important edge
cases.

## Non-Goals

- Do not reintroduce or import a `jjw-workspace-report` order just to satisfy
  disabled overrides.
- Do not remove the disabled override entries from
  `/data/projects/doltlite-gascity/city.toml`; keeping tombstones valid is the
  desired behavior for optional pack orders.
- Do not silence enabled missing-order override diagnostics.
- Do not make all missing overrides non-fatal at the `orders.ApplyOverrides`
  layer. Only disabled-only tombstones for globally absent order names should
  be ignored.
- Do not change order scanning, scheduling, patrol timing, or active-order
  filtering semantics beyond this override validation rule.

## Risk, Migration, And Rollback

- Primary behavior risk: accidentally silencing enabled missing-order overrides
  or mutating overrides. Mitigation: keep paired negative tests for enabled,
  nil-enabled, and field-mutating missing overrides.
- Scope risk: treating a disabled override for an existing order name but wrong
  rig as a tombstone would hide configuration mistakes. Mitigation: the helper
  must check all discovered orders by name before allowing the tombstone path.
- Public interface impact: no new CLI flags, API fields, schema fields, or
  config keys are planned. The user-facing change is narrower stderr/log
  behavior for disabled-only missing-order tombstones plus documentation.
- Migration impact: none. Existing `city.toml` disabled override entries remain
  valid and should not be removed as part of this fix.
- Rollback: revert the `internal/orders/override.go` helper and the associated
  tests/docs change. There is no data migration or runtime state to unwind.
- Artifact risk: the workflow root still records a missing generated
  requirements artifact at
  `/data/projects/doltlite-gascity/gascity/plans/order-override-missing-order-fix/build/requirements.md`.
  This plan remains traceable through `investigation-input.md` and the explicit
  coverage table above; the artifact mismatch should be treated as workflow
  bookkeeping debt, not as a reason to broaden the implementation.

## Verification

Run focused tests first:

```bash
go test ./internal/orders -run TestApplyOverrides
go test ./cmd/gc -run 'TestOrderScanContract(DisabledMissingOverrideIsSilent|EnabledMissingOverrideStillWarns|OverrideEnabledFalseMarksOrderDisabled)|TestBuildOrderDispatcherOverrideNotFoundNonFatal'
```

Then run the required repository gate:

```bash
go vet ./...
```

Manual verification for the current city:

```bash
gc order list
```

The command path should not emit `order "jjw-workspace-report" (rig "gascity")
not found` for disabled missing overrides. A temporary enabled or mutating
missing override in a test city should still produce a clear `not found`
diagnostic.

## GSTACK REVIEW REPORT

| Run | Status | Findings |
| --- | --- | --- |
| Design scope | Pass | No screen or component UI scope. The user-facing design surface is CLI/log behavior and order override documentation. |
| Requirements traceability | Pass with note | `REQ-001` through `REQ-004` map to the plan coverage table and task boundary table. The generated requirements artifact path is missing and remains captured as artifact debt. |
| Task boundaries | Pass | The plan now separates unit tests, command contract tests, override implementation, and docs into decomposition-ready beads. |
| Test strategy | Pass | The plan names focused `go test` commands and the repository `go vet ./...` gate. |
| Risk and rollback | Pass | The plan now records behavior risks, public-interface impact, migration impact, and rollback. |

VERDICT: pass - ready for decomposition after these plan amendments.

NO UNRESOLVED DECISIONS
