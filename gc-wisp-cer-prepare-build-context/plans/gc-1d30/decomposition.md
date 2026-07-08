---
schema: gc.build.decomposition.v1
workflow:
  id: gc-wisp-3io
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
    - path: /data/projects/doltlite-gascity/gascity/gc-wisp-cer-prepare-build-context/plans/gc-1d30/requirements.md
      hash: sha256:bc182ab547fa7e0986309daee02e43e5a66052bba4b4104bc21b2b1002144f30
    - path: /data/projects/doltlite-gascity/gascity/gc-wisp-cer-prepare-build-context/plans/gc-1d30/implementation-plan.md
      hash: sha256:917e5e54561f5e1a654c71dd07ae0daf4b8df4740dd65ad327987732874ea251
  coverage:
    - id: REQ-001
      status: covered
    - id: REQ-002
      status: covered
    - id: REQ-003
      status: covered
    - id: REQ-004
      status: covered
    - id: REQ-005
      status: covered
    - id: REQ-006
      status: covered
    - id: REQ-007
      status: covered
    - id: REQ-008
      status: covered
    - id: REQ-009
      status: covered
    - id: REQ-010
      status: covered
---

# Decomposition: Plugin-Safe Raw SQL for DoltLite Beads

## Summary

This decomposition creates a six-item implementation convoy for plugin-backed raw SQL support in Beads and the DoltLite backend plugin. The work preserves the existing direct `storage.RawDBAccessor` path for non-plugin stores while adding a typed plugin protocol path, pluginprocess client/store support, CLI routing and rendering, DoltLite backend plugin execution, mutation-boundary validation, and focused verification.

## Selected Downstream Formulas

| Purpose | Formula | Target |
| --- | --- | --- |
| Implement the convoy work items | implement | gc.implementation-worker |
| Execute individual implementation items | do-work-item | gc.implementation-worker |
| Review implementation output | review | agent review mode |
| Fix review findings if needed | fix-loop-base | gc.implementation-worker |

## Implementation Convoy

| Field | Value |
| --- | --- |
| Convoy ID | gc-pduh |
| Convoy name | raw-sql-plugin-implementation |
| Work items | gc-0yb1, gc-3thl, gc-6qcm, gc-1934, gc-mpnx, gc-fnrw |
| Source workflow | gc-wisp-3io |

`gc convoy create raw-sql-plugin-implementation gc-0yb1 gc-3thl gc-6qcm gc-1934 gc-mpnx gc-fnrw --json` returned `ok: true` with convoy ID `gc-pduh`.

Verification note: the required `gc convoy list --json` command failed after creation with `doltlite: validate schema: reading schema_migrations version: no such table: schema_migrations` while listing wisps. The created work beads are visible through `bd show`, and the workflow root metadata below records `gc-pduh` for downstream implementation dispatch.

## Work Items

| Bead ID | Title | Depends On | Requirement Trace | Plan Trace |
| --- | --- | --- | --- | --- |
| gc-0yb1 | Extend Beads plugin raw SQL protocol | None | REQ-001, REQ-002, REQ-003, REQ-006, REQ-007 | Proposed Implementation #1 |
| gc-3thl | Implement pluginprocess raw SQL client and store support | gc-0yb1 | REQ-001, REQ-002, REQ-003, REQ-004, REQ-008 | Proposed Implementation #2 |
| gc-6qcm | Route bd sql through plugin raw SQL when configured | gc-3thl | REQ-001, REQ-003, REQ-006, REQ-007, REQ-008 | Proposed Implementation #3 |
| gc-1934 | Implement DoltLite backend plugin raw SQL handler | gc-0yb1 | REQ-004, REQ-005, REQ-006, REQ-007, REQ-008 | Proposed Implementation #4, Proposed Implementation #6 |
| gc-mpnx | Preserve DoltLite mutation durability and backend boundaries | gc-1934 | REQ-005, REQ-007, REQ-008, REQ-010 | Proposed Implementation #5 |
| gc-fnrw | Add focused raw SQL tests and plugin smoke coverage | gc-3thl, gc-6qcm, gc-1934, gc-mpnx | REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006, REQ-007, REQ-008, REQ-009, REQ-010 | Verification, GSTACK REVIEW REPORT |

### gc-0yb1: Extend Beads plugin raw SQL protocol

Add typed raw SQL request/response support to `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture/backend/plugin/protocol.go`. Keep SQL opaque in the protocol, preserve deterministic column order, and represent mutation results without fabricating row columns.

Acceptance criteria:

- `RawSQLParams` includes `session_id` and `query`, plus only a read/write hint if the caller already computes one.
- `RawSQLResult` includes columns, keyed rows, optional `rows_affected`, and an explicit read/mutation indicator.
- JSON shape supports table, JSON, and CSV rendering without losing column order.
- Protocol changes preserve existing non-raw plugin operations.

### gc-3thl: Implement pluginprocess raw SQL client and store support

Wire `raw_sql` through `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture/internal/backend/pluginprocess/client.go` and `store.go`. The pluginprocess path checks hello capabilities, sends the active session id and query, preserves backend/transport error shape, and exposes raw SQL to the CLI without requiring `storage.RawDBAccessor` for plugin-backed stores.

Acceptance criteria:

- Client checks `RawSQL` capability before sending `raw_sql` and returns a distinct unsupported-capability error.
- `Store.RawSQL` sends the active session id and original query text.
- Backend and transport errors keep contextual method-failed details.
- Existing direct non-plugin `RawDBAccessor` behavior remains unchanged.

### gc-6qcm: Route bd sql through plugin raw SQL when configured

Update `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture/cmd/bd/sql.go` so plugin-backed scopes use the plugin raw SQL path while direct stores that implement `storage.RawDBAccessor` keep the existing direct path. Keep SQL classification at the CLI/rendering edge unless already needed by the backend.

Acceptance criteria:

- Plugin-backed storage is detected from configured store/backend state, not guessed from user input.
- Direct `storage.RawDBAccessor` path still works for non-plugin stores.
- Read query rendering continues to support table, JSON, and CSV.
- Mutation responses render consistently without requiring row columns.

### gc-1934: Implement DoltLite backend plugin raw SQL handler

Add `raw_sql` support in `/data/projects/doltlite-gascity/rigs/beads-backend-doltlite-plugin`, including the local `backend/plugin/protocol.go` compatibility copy, server dispatch, and provider/storage execution path. Execute SQL through the opened DoltLite plugin store or session, not by bypassing plugin lifecycle.

Acceptance criteria:

- Plugin protocol copy includes the same raw SQL request/response types as the client workspace.
- Server dispatch handles `raw_sql` with unknown-session and bad-SQL errors.
- Read-style SQL uses `QueryContext` and returns deterministic columns plus keyed rows.
- Mutation SQL uses `ExecContext` and returns `rows_affected` when available.
- `BackendCapabilities().RawSQL` is true only when the handler/provider path is wired and usable.

### gc-mpnx: Preserve DoltLite mutation durability and backend boundaries

Confirm mutation raw SQL behavior stays inside the DoltLite backend plugin boundary and matches existing direct `bd sql` semantics. If Dolt version commits are needed for durability, implement them using the existing transaction/commit conventions in `internal/storage/doltlite/transaction.go`; otherwise document the intentionally preserved direct-SQL semantics in focused tests.

Acceptance criteria:

- Raw SQL mutations use the opened plugin session and do not bypass lifecycle, locking, or telemetry.
- Mutation durability matches existing typed or direct raw SQL conventions, with any commit behavior implemented in the DoltLite storage layer.
- Focused tests document whether raw mutations create commits/events or intentionally preserve direct `bd sql` behavior.
- Capability reporting remains truthful for any build-tag or variant-specific limitations.

### gc-fnrw: Add focused raw SQL tests and plugin smoke coverage

Add focused verification for plugin-backed raw SQL across both workspaces. Cover pluginprocess client/store behavior, `cmd/bd` SQL rendering/routing, DoltLite plugin provider/server behavior, and a smoke path that builds compatible binaries and runs one read query plus one mutation query through `bd sql` with `backend_plugin_command`.

Acceptance criteria:

- In `beads-plugin-architecture`, focused tests cover pluginprocess `raw_sql` success rows, `rows_affected`, unsupported capability, backend errors, `Store.RawSQL` session/query forwarding, `cmd/bd` JSON/table/CSV reads, mutation rendering, and direct `RawDBAccessor` regression.
- In `beads-backend-doltlite-plugin`, focused tests cover `SELECT` or `PRAGMA` columns/rows, `UPDATE` or equivalent `rows_affected`, protocol response shape, unknown session, and bad SQL.
- Smoke script near `scripts/smoke-core-adapter.sh` proves plugin-backed `bd sql` read and mutation behavior when updated.
- Verification uses focused commands only and does not run broad `cmd/bd` package tests.

## Coverage

| ID | Status |
| --- | --- |
| REQ-001 | covered |
| REQ-002 | covered |
| REQ-003 | covered |
| REQ-004 | covered |
| REQ-005 | covered |
| REQ-006 | covered |
| REQ-007 | covered |
| REQ-008 | covered |
| REQ-009 | covered |
| REQ-010 | covered |
