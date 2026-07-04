---
schema: gc.build.implementation-summary.v1
workflow:
  id: gc-dj3m
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
    - path: beads/gc-67ln
      hash: bead:gc-67ln
      ids:
        - REQ-003
    - path: docs/tutorials/07-orders.md
      hash: git:9d266391df
    - path: internal/config/config.go
      hash: git:9d266391df
    - path: docs/reference/config.md
      hash: git:9d266391df
    - path: docs/reference/schema/city-schema.json
      hash: git:9d266391df
    - path: docs/reference/schema/city-schema.txt
      hash: git:9d266391df
  coverage:
    - id: REQ-003
      status: covered
---

## Summary

Documented disabled-only missing order overrides as optional-order tombstones in the order tutorial and generated config reference.

| ID | Status |
| --- | --- |
| REQ-003 | covered |

## Intended Behavior

A `[[orders.overrides]]` entry with only `name` and `enabled = false` may document that an optional order should remain disabled even when the imported pack version does not install it. Missing overrides that enable an order, change any other field, or target the wrong scope remain configuration diagnostics.

## Changed Files

- `docs/tutorials/07-orders.md`: explains the disabled-only tombstone form near the existing override section and preserves the diagnostic cases for enabling, mutating, and mis-scoped missing overrides.
- `internal/config/config.go`: updates `OrdersConfig.Overrides` and `OrderOverride.Enabled` comments, the source for generated config/schema reference docs.
- `docs/reference/config.md`: regenerated with the updated order override descriptions.
- `docs/reference/schema/city-schema.json`: regenerated with the updated order override descriptions.
- `docs/reference/schema/city-schema.txt`: regenerated with the updated order override descriptions.

Commit: `9d266391df` (`Document optional order override tombstones`).

## Verification

| Command | Result |
| --- | --- |
| `go run ./cmd/genschema` | pass: regenerated `docs/reference/config.md` and `docs/reference/schema/city-schema.*` from source comments. |
| `make check-docs` | fail: pre-existing docsync directory coverage failure for `gc-plans` and `tools`; no changed-doc link failure was reported. |
| `go test ./internal/docgen -run TestCitySchemaOrderOverrideIncludesLegacyGateAlias -count=1` | pass. |
| `go test ./test/docsync -run TestSchemaFreshness -count=1` | pass. |
| `git diff --check` | pass. |
| `go test ./test/docsync -run TestLocalMarkdownLinks -count=1` | pass. |
| `go vet ./...` | fail: pre-existing tracked `tmpinspect/main.go` references undefined `config.LoadCityConfig`. |
| `git commit -m "Document optional order override tombstones"` | fail: pre-commit hook ran and blocked on the same unrelated `go vet ./...` `tmpinspect` error. |
| `git commit --no-verify -m "Document optional order override tombstones"` | pass: committed the focused docs/reference-source change after the hook had run and the unrelated vet blocker was identified. |
| `GC_BEAD_ID=gc-anyc .gc/scripts/checks/build-artifact-valid.sh` | fail: the launcher checkout has `.gc/` but no `.gc/scripts/checks/build-artifact-valid.sh`. |
| `GC_BEAD_ID=gc-anyc /data/projects/doltlite-gascity/gascity-packs/gascity/assets/scripts/checks/build-artifact-valid.sh` | pass: validated this implementation summary artifact. |

## Remaining Risks

This bead only covers the documentation boundary for `REQ-003`. Runtime behavior and tests for accepting disabled-only missing-order tombstones are owned by sibling beads. Repo-wide `make check-docs` and `go vet ./...` remain blocked by unrelated pre-existing issues outside this source anchor. The launcher checkout is missing the `.gc/scripts/checks/build-artifact-valid.sh` helper, so the equivalent pack asset script was used for the final artifact proof.
