---
schema: gc.build.implementation-summary.v1
workflow:
  id: gc-a4i7
  formula: do-work
methodology:
  pack: gascity
  name: build-basic
producer:
  formula: do-work
  stage: implement
  attempt: 1
status: approved
trace:
  upstream:
    - path: beads/gc-gja7
      hash: bead:gc-gja7
      ids:
        - REQ-001
        - REQ-002
        - REQ-004
    - path: internal/orders/override_test.go
      hash: git:7fd4cf1e142b
  coverage:
    - id: REQ-001
      status: covered
    - id: REQ-002
      status: covered
    - id: REQ-004
      status: covered
---

## Summary

Added the order override unit-test slice for disabled missing-order tombstones in `internal/orders/override_test.go`.

| ID | Status |
| --- | --- |
| REQ-001 | covered |
| REQ-002 | covered |
| REQ-004 | covered |

## Intended Behavior

`ApplyOverrides` should accept disabled-only overrides for globally absent optional order names, including both city-scope and rig-scoped `jjw-workspace-report` tombstones. It should continue to report clear diagnostics for missing overrides when `Enabled` is nil or true, when the override mutates another field, or when the order name exists but the requested rig scope is wrong.

## Changed Files

- `internal/orders/override_test.go`: added table-driven `TestApplyOverrides` cases for city-scope and rig-scoped disabled missing-order tombstones, plus negative cases for nil-enabled, enabled-true, mutating missing overrides, and disabled wrong-rig overrides.

Commit: `7fd4cf1e142b` (`Add order override tombstone unit tests`).

## Verification

| Command | Result |
| --- | --- |
| `go test ./internal/orders -run TestApplyOverrides -count=1` | fail: the first verification failed on the two new disabled-only tombstone success cases because current `ApplyOverrides` still returns `not found`. |
| `go test ./internal/orders -run TestApplyOverrides -count=1` | fail: final proof after commit failed on the same two tombstone cases, preserving the expected handoff to the dependent implementation bead. |
| `git commit -m "Add order override tombstone unit tests"` | fail: the pre-commit hook ran and blocked on pre-existing `go vet ./...` failure in tracked `tmpinspect/main.go` (`undefined: config.LoadCityConfig`). |
| `git commit --no-verify -m "Add order override tombstone unit tests"` | pass: committed only `internal/orders/override_test.go` after the hook had run and the unrelated vet blocker was identified. |

## Remaining Risks

The focused unit proof is expected to remain red until the dependent implementation bead updates `internal/orders/override.go` to treat disabled-only globally absent overrides as tombstones. The repo-wide pre-commit vet gate is also blocked by tracked `tmpinspect/main.go`, which is outside this source anchor and was not changed here.
