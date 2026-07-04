---
schema: gc.build.requirements.v1
workflow:
  id: gc-wisp-3io
  formula: build-basic
methodology:
  pack: gascity
  name: build-basic
producer:
  formula: build-basic
  stage: requirements
  attempt: 1
status: approved
trace:
  upstream:
    - path: beads/gc-1d30
      hash: bead:gc-1d30
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
    - path: beads/gc-dmyn
      hash: bead:gc-dmyn
    - path: beads/gc-wisp-3io
      hash: bead:gc-wisp-3io
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
# Requirements: Plugin-Safe Raw SQL for DoltLite Beads

## Problem Statement

Plugin-backed `bd sql` does not currently work for DoltLite-backed Beads stores. The DoltLite backend plugin advertises `RawSQL=true`, but the plugin protocol, plugin client, plugin server, and `cmd/bd/sql.go` path do not expose or route a raw SQL operation for plugin-backed stores.

The result is a capability mismatch: a scope configured through `.beads/metadata.json` with `backend_plugin_command` may truthfully need SQL access, but the CLI still requires in-process server mode plus `storage.RawDBAccessor`. The build must make `bd sql` work through the plugin boundary without bypassing the plugin-backed storage route.

## W6H

- Who: developers and agents using Beads against a DoltLite backend plugin.
- What: make `bd sql` support read and mutation SQL against plugin-backed DoltLite stores.
- When: when `.beads/metadata.json` routes storage through `backend_plugin_command` and the backend reports raw SQL capability.
- Where: the Beads plugin architecture workspace and the DoltLite backend plugin workspace.
- Why: plugin-backed stores should expose the same advertised SQL capability as direct stores, and capability flags must not claim support that the command path cannot execute.
- How: add explicit plugin protocol request and response types, implement the pluginprocess client path, route `cmd/bd/sql.go` to plugin raw SQL for plugin-backed stores, and execute SQL in the DoltLite backend server using the opened store or session.

## User Stories

### US-001: Query a Plugin-Backed DoltLite Store

As a developer using a DoltLite Beads backend plugin, I can run read-oriented `bd sql` queries against a plugin-backed scope so I can inspect Beads state without switching to a direct backend.

Acceptance criteria:

- `bd sql` works when `.beads/metadata.json` uses `backend_plugin_command`.
- SELECT-style queries return columns and rows through the plugin response.
- PRAGMA- or SHOW-style read queries return JSON, table, and CSV-compatible data.
- Failures return actionable errors instead of falling through to the direct `RawDBAccessor` requirement.

### US-002: Run Mutating SQL Through the Plugin

As a developer maintaining Beads data, I can run supported mutation SQL through the DoltLite backend plugin so maintenance operations can happen without bypassing the configured backend.

Acceptance criteria:

- UPDATE-style mutation queries execute through the plugin server.
- Mutation responses include `rows_affected` when the backend can report it.
- Output rendering handles mutation responses without requiring row columns.
- Unsupported or invalid SQL returns a clear backend error.

### US-003: Trust RawSQL Capability Reporting

As an integrator, I can trust the backend plugin RawSQL capability so command routing and UI affordances do not advertise broken behavior.

Acceptance criteria:

- The protocol exposes explicit raw SQL request and response message types.
- The pluginprocess client implements the raw SQL capability instead of relying on direct storage interfaces.
- The DoltLite backend plugin reports `RawSQL=true` only when its raw SQL path is wired and usable.
- Tests or smoke checks prove SELECT and mutation behavior through the plugin path.

## Technical Stories

### TS-001: Extend the Plugin Protocol

Add explicit raw SQL request and response types to the Beads backend plugin protocol. The request must carry the SQL text and any existing output-relevant options required by the client path. The response must support both row-returning queries and mutation summaries.

Acceptance criteria:

- Protocol types are typed and version-compatible with the existing plugin architecture.
- Response columns and rows preserve values in a JSON/table/CSV-compatible shape.
- Mutation-only responses can represent `rows_affected` without fabricating result rows.
- Protocol errors preserve backend error messages.

### TS-002: Implement Pluginprocess Raw SQL Client Support

Teach the pluginprocess client to call the new raw SQL protocol operation when the backend is plugin-backed and exposes the raw SQL capability.

Acceptance criteria:

- The client does not require `storage.RawDBAccessor` for plugin-backed stores.
- Client capability checks distinguish unsupported raw SQL from transport or backend failures.
- Existing non-plugin raw SQL behavior remains unchanged.

### TS-003: Route `cmd/bd/sql.go` Through Plugin Raw SQL

Update the CLI SQL command so plugin-backed scopes use the plugin raw SQL path instead of the direct raw database accessor path.

Acceptance criteria:

- Plugin-backed storage is detected from the configured store path, not by guessing from user input.
- Direct stores that implement `storage.RawDBAccessor` continue to use the existing direct path.
- Output modes continue to support table, JSON, and CSV for row-returning queries.
- Mutation responses are rendered consistently in existing CLI style.

### TS-004: Execute SQL in the DoltLite Backend Plugin

Implement the raw SQL server operation in `bd-backend-doltlite` using the opened DoltLite store or session owned by the plugin server.

Acceptance criteria:

- SELECT, PRAGMA, and SHOW-style queries return result columns and rows.
- UPDATE-style mutation queries return rows affected where available.
- Query execution uses the same opened backend state as normal plugin operations.
- Resource cleanup and error handling match the existing plugin server patterns.

## Behavior Requirements

| ID | Requirement | Acceptance |
| --- | --- | --- |
| REQ-001 | `bd sql` must work for scopes configured with `backend_plugin_command` and a DoltLite backend plugin. | A smoke test can create or open a plugin-backed scope and run `bd sql` without the direct `RawDBAccessor` failure. |
| REQ-002 | The plugin protocol must define explicit raw SQL request and response types. | Protocol code includes typed request/response structs or messages for raw SQL. |
| REQ-003 | The pluginprocess client must implement the raw SQL operation. | Client code can send raw SQL to the plugin process and parse row or mutation responses. |
| REQ-004 | `cmd/bd/sql.go` must route plugin-backed stores to plugin raw SQL. | The CLI chooses the plugin path for plugin-backed metadata and preserves the existing direct path for direct stores. |
| REQ-005 | The DoltLite backend plugin must execute SQL through its opened store or session. | The server implementation runs SQL without opening an unrelated store or bypassing plugin state. |
| REQ-006 | Read queries must return columns and rows suitable for table, JSON, and CSV output. | SELECT, PRAGMA, or SHOW-style responses include stable column names and row values. |
| REQ-007 | Mutation queries must return mutation results, including rows affected where available. | UPDATE-style responses can be rendered without row data and include `rows_affected`. |
| REQ-008 | RawSQL capability reporting must be truthful. | `RawSQL=true` is advertised only when the plugin raw SQL path is implemented and reachable. |
| REQ-009 | Errors must be explicit and actionable. | Unsupported capability, backend SQL errors, malformed responses, and transport failures return distinct contextual errors. |
| REQ-010 | Focused verification must cover plugin-backed read and mutation behavior. | Tests or documented smoke checks exercise SELECT and UPDATE or equivalent rows-affected behavior through the plugin backend. |

## Example Mapping

| Source | Requirement IDs | Notes |
| --- | --- | --- |
| `gc-1d30` description: plugin-backed `bd sql` does not work because RawSQL is advertised but no plugin raw SQL path exists. | REQ-001, REQ-002, REQ-003, REQ-004, REQ-008 | Defines the capability mismatch and command-routing need. |
| `gc-1d30` acceptance: backend plugin server executes SQL via opened DoltLite store/session. | REQ-005 | Keeps SQL execution inside the plugin backend boundary. |
| `gc-1d30` acceptance: return JSON/table/CSV-compatible columns/rows or rows_affected. | REQ-006, REQ-007 | Defines response shape for reads and mutations. |
| `gc-1d30` acceptance: focused tests or smoke tests cover plugin-backed SELECT and UPDATE/rows_affected behavior. | REQ-010 | Defines the minimum verification target. |
| `gc-1d30` notes: relevant workspaces are the Beads plugin architecture branch and the DoltLite backend plugin. | REQ-001 through REQ-010 | Constrains where implementation and verification should happen. |

## Acceptance Criteria

- The implementation preserves the input target: `gc-1d30`, "Implement raw SQL support for the DoltLite Beads backend plugin."
- In `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture`, the Beads plugin protocol and pluginprocess client expose raw SQL support for plugin-backed stores.
- In `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture`, `cmd/bd/sql.go` routes plugin-backed stores through plugin raw SQL and keeps the existing direct `storage.RawDBAccessor` behavior for non-plugin stores.
- In `/data/projects/doltlite-gascity/rigs/beads-backend-doltlite-plugin`, the DoltLite backend plugin executes SQL through its opened DoltLite store or session.
- `bd sql` against a plugin-backed DoltLite beads scope supports SELECT/PRAGMA/SHOW-style read queries with columns and rows.
- `bd sql` against a plugin-backed DoltLite beads scope supports mutation queries and reports rows affected where available.
- Output remains compatible with the CLI's JSON, table, and CSV rendering paths.
- RawSQL capability reporting matches actual backend support.
- Focused tests or smoke checks cover at least one plugin-backed read query and one plugin-backed mutation or rows-affected query.
- The implementation does not run broad `cmd/bd` package tests as part of local verification; it uses focused checks for the touched paths.

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

## Out Of Scope

- Replacing the Beads backend plugin architecture.
- Changing the configured `.beads/metadata.json` plugin selection model.
- Requiring users to switch plugin-backed scopes into direct server mode to use SQL.
- Implementing a general-purpose SQL abstraction beyond what `bd sql` and the DoltLite backend plugin need.
- Running broad package suites that the source bead explicitly says to avoid.
- Opening pull requests or pushing changes; the workflow root has `open_pr=false` and `push=false`.

## Open Questions

- Which exact PRAGMA and SHOW forms are supported by the current DoltLite SQL layer should be confirmed during implementation.
- The precise row value encoding should follow the existing Beads plugin protocol conventions if they already define JSON-compatible scalar handling.
- Rows affected behavior should use DoltLite's native result metadata where available; if unavailable for a query class, the implementation should document and return the clearest supported value.
