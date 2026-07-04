---
schema: gc.build.plan.v1
workflow:
  id: gc-wisp-3io
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
    - path: /data/projects/doltlite-gascity/gascity/gc-wisp-cer-prepare-build-context/plans/gc-1d30/requirements.md
      hash: sha256:bc182ab547fa7e0986309daee02e43e5a66052bba4b4104bc21b2b1002144f30
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
        - REQ-005
        - REQ-006
        - REQ-007
        - REQ-008
        - REQ-009
        - REQ-010
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

# Implementation Plan: Plugin-Safe Raw SQL for DoltLite Beads

## Summary

Add raw SQL support to the plugin-backed Beads path without weakening the existing direct-store behavior. The implementation spans two ownership boundaries:

- `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture` owns the public backend plugin protocol, the pluginprocess client/store adapter, and `cmd/bd sql` routing/rendering.
- `/data/projects/doltlite-gascity/rigs/beads-backend-doltlite-plugin` owns the DoltLite backend plugin server and the execution of SQL against the already-opened DoltLite store/session.

The target behavior is that `bd sql` against a `.beads/metadata.json` configured with `backend_plugin_command` sends a typed `raw_sql` request to the plugin process, receives either rows or a mutation summary, and renders the result through the same JSON/table/CSV user surfaces that direct `storage.RawDBAccessor` stores already use. Direct stores continue to use the existing `*sql.DB` path.

## Current System

In the Beads workspace, `backend/plugin/protocol.go` already advertises `Capabilities.RawSQL`, but it does not define a raw SQL request or response operation. `internal/backend/pluginprocess/client.go` has a generic request helper over line-oriented JSON RPC, and `internal/backend/pluginprocess/store.go` wraps plugin sessions as `storage.DoltStorage`, but the wrapper does not expose raw SQL.

`cmd/bd/sql.go` currently rejects embedded mode, requires a global `store`, unwraps it, type-asserts `storage.RawDBAccessor`, then calls `UnderlyingDB().QueryContext` for read-style SQL and `ExecContext` for mutations. The command renders reads as JSON, CSV, or table rows, and renders mutations as `rows_affected` JSON or a text count. Plugin-backed stores created in `cmd/bd/store_factory.go` go through `pluginprocess.Open`, so they cannot satisfy `RawDBAccessor` and fail before reaching any plugin capability.

In the DoltLite backend plugin, `cmd/bd-backend-doltlite/main.go` serves the same plugin protocol and dispatches request methods to `internal/provider.Manager` and `provider.Session`. The provider advertises `RawSQL: true` in `internal/provider/provider.go`, but there is no request handler for raw SQL. The owned DoltLite store already holds a persistent `*sql.DB` in `internal/storage/doltlite/store.go` and exposes `DB()`/`UnderlyingDB()`, while `OpenSQL` centralizes DSN setup in `internal/storage/doltlite/open.go`.

## Proposed Implementation

1. Extend the Beads backend plugin protocol.

   Add typed protocol structs in `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture/backend/plugin/protocol.go`:

   - `RawSQLParams` with `session_id`, `query`, and a read/write mode hint only if the caller already computes one for output behavior. Keep the SQL text opaque to the protocol; classification remains a CLI rendering concern unless the backend needs it for execution.
   - `RawSQLResult` with `columns []string`, `rows []map[string]any`, optional `rows_affected`, and an explicit read/mutation flag.
   - A mutation result must be representable without fabricating columns or rows.

   Prefer a stable JSON shape that is easy for table, JSON, and CSV output to consume. Preserve a separate `columns` slice so CSV/table order is deterministic even though row values are keyed maps.

2. Teach pluginprocess to call `raw_sql`.

   In `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture/internal/backend/pluginprocess/client.go`, add a method that checks the hello capabilities before sending `raw_sql`. Unsupported capability should return a distinct contextual error such as `backend plugin does not support raw SQL`; transport and backend execution errors should preserve the existing `method failed: code: message` shape.

   In `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture/internal/backend/pluginprocess/store.go`, expose a small raw SQL capability on `Store`. The least disruptive option is a new internal interface such as:

   ```go
   type RawSQLExecutor interface {
       RawSQL(ctx context.Context, query string) (backendplugin.RawSQLResult, error)
   }
   ```

   `pluginprocess.Store` should satisfy that interface by sending `RawSQLParams{SessionID: s.sessionID, Query: query}`. This keeps plugin raw SQL explicit and avoids pretending that a plugin-backed store has a local `*sql.DB`.

3. Route `bd sql` through plugin raw SQL when available.

   Update `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture/cmd/bd/sql.go` to split execution from rendering:

   - Keep the existing direct path for stores satisfying `storage.RawDBAccessor`.
   - Before returning `storage backend does not support raw DB access`, check whether `storage.UnwrapStore(store)` or `store` satisfies the new plugin raw SQL interface.
   - For plugin-backed stores, call `RawSQL(ctx, query)` and pass the returned result to shared render helpers.
   - Preserve current read classification for deciding whether JSON output should be a row array or a mutation object if the protocol response does not make that explicit. Prefer making the response explicit enough that rendering can branch on `len(columns) > 0` or `rows_affected != nil`.

   Extract existing table/JSON/CSV rendering into helpers inside `cmd/bd/sql.go` or a focused sibling file, keeping behavior unchanged for direct stores. CSV should still write the header from `columns`; mutation results should not try to emit row CSV.

4. Implement server-side raw SQL in the DoltLite backend plugin.

   In `/data/projects/doltlite-gascity/rigs/beads-backend-doltlite-plugin/cmd/bd-backend-doltlite/main.go`, add a `case "raw_sql"` handler that decodes `backendplugin.RawSQLParams`, fetches the session from the provider manager, calls a provider method, and returns `RawSQLResult`. Error codes should distinguish bad params, unknown session, and SQL execution failure.

   In `/data/projects/doltlite-gascity/rigs/beads-backend-doltlite-plugin/internal/provider/provider.go`, add `Session.RawSQL(ctx, query string)`. The method should use the session's opened `s.Store` and its persistent `UnderlyingDB()` or `DB()` connection instead of opening an unrelated store. Implement query execution in one place, either in provider or a small storage helper:

   - For read-style SQL (`SELECT`, `EXPLAIN`, `PRAGMA`, `SHOW`), run `QueryContext`, read columns, scan row values, convert `[]byte` to string, and return deterministic columns plus row values.
   - For mutation SQL, run `ExecContext` and return `rows_affected` when `RowsAffected` succeeds.
   - Return contextual backend errors for query, column, scan, row iteration, and exec failures.

   Keep `BackendCapabilities().RawSQL` truthful. If raw SQL execution is guarded by build tags or missing in a variant, report `RawSQL: false` for that variant; otherwise keep it true only after the handler and provider method are wired.

5. Keep mutations inside the backend boundary.

   The plugin raw SQL path must use the opened plugin session and must not bypass plugin lifecycle, locking, or telemetry. If Dolt version commits are required for raw mutation SQL to be durable in the same way as typed mutations, implement that in the DoltLite plugin storage layer using the existing transaction/commit conventions from `internal/storage/doltlite/transaction.go`. If current direct `bd sql` semantics intentionally execute raw mutations without creating a typed storage event or commit, preserve that behavior and document it in the focused tests.

6. Update compatibility copies deliberately.

   The DoltLite backend plugin has its own `backend/plugin/protocol.go` copy. Port the new raw SQL protocol types into that copy in the same change set as the server handler so client and plugin binaries agree on JSON names. Avoid changing unrelated protocol fields.

## Non-Goals

- Do not replace `storage.RawDBAccessor` for direct Dolt or SQLite-backed stores.
- Do not add a generic local `*sql.DB` accessor to pluginprocess stores.
- Do not build SQL parsing or policy enforcement beyond the existing read/mutation classification needed for rendering.
- Do not broaden `bd sql` support to embedded mode unless it is already supported by the current command architecture.
- Do not change Beads issue schema, migrations, or normal typed CRUD behavior.
- Do not run broad `cmd/bd` package tests locally as part of this work.

## Verification

Add focused tests in `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture`:

- `internal/backend/pluginprocess/client_test.go`: a fake plugin process supports `raw_sql`, returns rows, returns `rows_affected`, reports unsupported capability, and propagates backend errors.
- `internal/backend/pluginprocess/store_test.go` or the existing client test file: `Store.RawSQL` sends the active session id and query.
- `cmd/bd/sql_test.go`: a plugin raw SQL executor path renders row results for JSON/table/CSV and renders mutation results without requiring columns; direct `storage.RawDBAccessor` behavior remains covered by existing helpers or a focused regression test.

Add focused tests or smoke checks in `/data/projects/doltlite-gascity/rigs/beads-backend-doltlite-plugin`:

- Provider-level raw SQL tests for `SELECT` or `PRAGMA` returning columns/rows and `UPDATE` or equivalent mutation returning rows affected.
- Server protocol test for `raw_sql` request/response shape and unknown-session or bad-SQL errors.
- A smoke script update, likely near `scripts/smoke-core-adapter.sh`, that builds compatible `bd` and plugin binaries, configures `backend_plugin_command`, then runs one read query and one mutation query through `bd sql`.

Run only focused commands:

```bash
go test ./internal/backend/pluginprocess ./cmd/bd -run 'Test.*RawSQL|TestSQL'
go test ./cmd/bd-backend-doltlite ./internal/provider -run 'Test.*RawSQL'
```

If the smoke script is updated, run that script directly after the focused Go tests. Do not run the full test suite locally.

Coverage matrix:

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

## GSTACK REVIEW REPORT

| Run | Status | Findings |
| --- | --- | --- |
| Design scope | pass | No UI, UX, dashboard, frontend, screen, page, component, or user-facing interaction scope was found in the implementation plan. Visual mockups are not applicable. |
| Requirements traceability | pass | The plan front matter and Markdown coverage table cover REQ-001 through REQ-010, matching the requirements artifact. |
| Task boundaries | pass | The proposed work decomposes cleanly into protocol, pluginprocess client/store, CLI routing/rendering, DoltLite plugin server/provider, mutation semantics, protocol-copy compatibility, and verification tasks. |
| Test strategy | pass | The plan names focused Go test packages and a smoke-script strategy for plugin-backed read and mutation SQL without requiring a broad local suite. |
| Risk and rollback | pass_with_note | Risky files, ownership boundaries, capability truthfulness, direct-store compatibility, protocol-copy sync, and mutation durability are explicit. Decomposition should preserve the existing direct RawDBAccessor path as the rollback boundary. |

VERDICT: PASS - ready for decomposition. The only plan edit made during review was to resolve the raw SQL row-shape ambiguity in favor of the existing keyed-row plus columns contract.

NO UNRESOLVED DECISIONS
