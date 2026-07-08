# Gas City DoltLite Integration Migration Map

This document maps what the Gas City fork currently does for DoltLite-backed
Beads storage and DoltLite fast paths, then describes how those responsibilities
should move as `beads-backend-doltlite` becomes both the Beads backend plugin
and the DoltLite-specific Gas City companion plugin.

The goal is not to remove every direct DoltLite use from Gas City. The goal is
to move backend/storage ownership into the plugin, keep Gas City responsible for
city orchestration, and leave only narrow, justified fast paths in Gas City.

## Current Pieces

### 1. Backend Selection In Gas City Core

Files:

- `cmd/gc/beads_backend.go`
- `cmd/gc/bd_env.go`
- `cmd/gc/beads_provider_lifecycle.go`
- `cmd/gc/init_provider_readiness.go`
- `cmd/gc/cmd_bd_store_bridge.go`

Current behavior:

- `BeadsBackend` abstracts backend identity for `dolt`, `doltlite`, and
  external backends.
- `doltliteBackend` declares:
  - no managed server;
  - no Dolt binary requirement;
  - no managed Dolt doctor checks;
  - provider env `GC_BEADS_BACKEND=doltlite` and `BEADS_BACKEND=doltlite`;
  - builtin bootstrap pack `beads-doltlite-init`.
- `resolveScopeBeadsBackend` and `scopeBackendIsDoltlite` decide whether a
  city or rig scope is DoltLite-backed from `city.toml`, env, and
  `.beads/metadata.json`.
- bd child process environments scrub managed Dolt/Postgres variables when the
  scope is DoltLite-backed.
- DoltLite-linked child launchers scrub ambient `LD_LIBRARY_PATH` and
  `DYLD_LIBRARY_PATH` so the process uses the pack-selected `libdoltlite`.

Migration target:

- Keep backend selection in Gas City. Gas City still needs to decide which city
  backend is active before it can launch packs, agents, commands, and init.
- Move DoltLite-specific metadata validation details into the plugin companion
  (`gc-doltlite health/layout`) and have Gas City call or reuse that contract.
- Keep environment projection in Gas City, but reduce the backend-specific
  logic to:
  - discover backend;
  - set `BEADS_BACKEND=doltlite`;
  - expose `backend_plugin_command` metadata;
  - scrub incompatible managed-server env.

### 2. Metadata And Init Contract

Files:

- `cmd/gc/beads_provider_lifecycle.go`
- `internal/beads/contract/files.go`
- `gascity-packs/beads-doltlite-init/assets/scripts/gc-beads-doltlite-bd.sh`
- `gascity-packs/beads-doltlite/assets/scripts/gc-beads-doltlite-bd.sh`

Current behavior:

- Gas City canonicalizes `.beads/metadata.json` for DoltLite scopes.
- Current DoltLite metadata includes:
  - `backend: "doltlite"`;
  - `database: "doltlite"`;
  - `dolt_database`;
  - `project_id` where available;
  - `attached_databases`, currently intended to include `ops`.
- `ensureCanonicalDoltliteScopeMetadata` writes `attached_databases` with
  alias `ops` and path `<scope>/.gc/ops.sqlite`.
- The init pack writes minimal metadata before the full external pack is
  installed.
- The full pack wrapper still writes older DoltLite metadata and bootstraps
  runtime config such as `issue_prefix` and `types.custom`.
- The init pack already knows how to write plugin fields when it can find
  `bd-backend-doltlite`:
  - `backend_plugin_command`;
  - `backend_plugin_args`, usually with trace and `serve`.

Migration target:

- The plugin should own the exact DoltLite metadata schema and validation.
- Gas City should still materialize metadata during `gc init`, because init has
  to work before Beads can reliably open the store.
- The metadata writer should converge on one shape:

```json
{
  "backend": "doltlite",
  "database": "doltlite",
  "dolt_database": "gascity",
  "backend_plugin_command": "/absolute/path/to/bd-backend-doltlite",
  "backend_plugin_args": ["--trace", "/path/to/trace.jsonl", "serve"],
  "attached_databases": [
    {"alias": "ops", "path": ".gc/ops.sqlite"}
  ],
  "gascity": {
    "doltlite_profile": "ledger"
  }
}
```

- The plugin companion already has the start of this contract in
  `internal/gcplugin/layout.go` and `cmd/gc-doltlite`.
- Next step: make both `beads-doltlite-init` and the full `beads-doltlite` pack
  use the same metadata writer, including `attached_databases` and plugin
  command fields.

### 3. Pack Install And Build Story

Files:

- `gascity-packs/beads-doltlite/pack.toml`
- `gascity-packs/beads-doltlite/commands/build/run.sh`
- `gascity-packs/beads-doltlite/commands/build/help.md`
- `gascity-packs/beads-doltlite/commands/health/run.sh`
- `gascity-packs/beads-doltlite/doctor/*`

Current behavior:

- `beads-doltlite` is the full runtime pack.
- `beads-doltlite-init` is the small builtin bootstrap pack used before
  external packs are installed.
- The build command installs coordinated DoltLite-linked artifacts:
  - `libdoltlite`;
  - `bd` / `bd-doltlite`;
  - optionally `doltlite-client`;
  - a DoltLite-linked `gc` binary with `gascity_doltlite_lib`.
- Doctor checks verify:
  - metadata says DoltLite;
  - health command works;
  - fast path responds;
  - `gc` is linked against `libdoltlite`;
  - `sqlite3` exists for legacy maintenance paths.
- Maintenance commands use `bd flatten` and `bd gc` under DoltLite env.

Migration target:

- The pack should install the backend plugin binary alongside `bd`, and write
  its absolute path into `.beads/metadata.json`.
- The pack should also install the optional `gc-doltlite` helper once it is
  promoted from scaffold to stable companion command.
- `sqlite3` should stop being part of the normal DoltLite schema/init story.
  Schema creation and raw SQL should go through the plugin/backend, not a
  separate SQLite CLI.
- Build alignment remains important: `bd`, `bd-backend-doltlite`, `gc`, and
  any diagnostic client must link against the same `libdoltlite` release or
  explicitly configured development build.

### 4. Gas City Native DoltLite Fast Path

Files:

- `cmd/gc/doltlite_store_native.go`
- `cmd/gc/doltlite_store_default.go`
- `cmd/gc/bd_env.go`
- `internal/beads/doltlite_read_store.go`
- `internal/beads/doltlite_count.go`
- `internal/beads/doltlite_lib_link.go`
- `internal/beads/doltlite_sqlite_driver_libdoltlite.go`

Current behavior:

- The native fast path is compiled only with `gascity_doltlite_lib`.
- `openOptimizedDoltliteStore` wraps a normal `BdStore` with
  `DoltliteReadStore` when the scope is DoltLite-backed.
- The fast path opens `.beads/doltlite/<dolt_database>.db` using the DoltLite
  SQLite driver in read-only mode for reads.
- It attaches databases declared in metadata, currently `ops`, but live Beads
  and Gas City table logic still targets main DoltLite tables.
- It bypasses the `bd` process for hot reads and selected writes.

Fast path reads:

- `issues`
- `wisps`
- `labels`
- `wisp_labels`
- `dependencies`
- `wisp_dependencies`

Fast path read operations:

- `Get`
- `GetSessionBead`
- `ListSessionBeads`
- `List`
- `ListOpen`
- `Children`
- `ListByLabel`
- `ListByAssignee`
- `ListByMetadata`
- `Ready`
- `LastOrderRun`
- `HasOpenOrderRun`
- `DepList`
- `DepListBatch`
- `Count` for supported query shapes

Fast path writes:

- durable `issues`:
  - update metadata/status/general fields;
  - delete;
  - label/dependency changes when the target bead is durable.
- `wisps`:
  - create;
  - update;
  - close/reopen;
  - delete;
  - metadata updates.
- `wisp_labels`:
  - insert/delete.
- `wisp_dependencies`:
  - insert/delete/update parent-child and blocker edges.

Important semantics:

- `wisps` can be `ephemeral` or `no_history`.
- `TierIssues` includes durable issues plus `no_history` wisps, excluding
  ephemeral rows.
- `TierWisps` reads ephemeral and no-history rows.
- `TierBoth` merges issues and wisps with de-duplication.
- Metadata filters use SQL `LIKE` only as a narrowing filter and then apply
  exact metadata matching in Go.
- Timestamp parsing accepts multiple DoltLite/SQLite formats.
- `currentDoltHash` is not a real Dolt commit hash. It is a table fingerprint
  from counts and max timestamps, used to invalidate caches.
- Writes use a `.bd-write.lock` lock and retry transient write errors.

Migration target:

- Move the table/query implementation into the plugin, because this is backend
  storage behavior. The plugin already has parallel packages:
  - `internal/storage/doltlite`;
  - `internal/storage/issueops`;
  - `internal/storage/sqlbuild`;
  - `internal/storage/versioncontrolops`;
  - `internal/storage/conformance`.
- Keep a narrow Gas City fast path only if it is a client of the plugin or uses
  a plugin-owned library package. Gas City should not carry its own second copy
  of DoltLite query semantics.
- If in-process speed is still required, expose a stable Go package from the
  plugin repo that Gas City can import behind `gascity_doltlite_lib`, rather
  than keeping `internal/beads/doltlite_read_store.go` as fork-only code.
- The first safe removal target is duplicated metadata/layout logic. Query
  semantics should move only after conformance/parity tests prove that Gas City
  ready/list/routing behavior matches.

### 5. ops.sqlite And Table Placement

Files:

- `cmd/gc/beads_provider_lifecycle.go`
- `internal/beads/contract/files.go`
- `internal/beads/doltlite_read_store.go`
- `gascity/plans/doltlite-ops-sqlite-table-audit.md`
- plugin: `internal/gcplugin/layout.go`

Current behavior:

- Gas City writes an `attached_databases` metadata entry for `ops`.
- The fast path attaches `ops.sqlite` when it opens a DoltLite DB.
- No current production table is actually routed to `ops.*`.

Known table-placement conclusions:

- Strong first candidates for `ops.sqlite`:
  - `repo_mtimes`;
  - `local_metadata`;
  - future diagnostics/metrics/lock-observation tables.
- Not safe to move casually:
  - `issues`;
  - `labels`;
  - `dependencies`;
  - `comments`;
  - `events`;
  - `config`;
  - `metadata`;
  - `routes`;
  - counters;
  - custom status/type tables;
  - compaction/history tables.
- Workflow/wisp tables are configuration-dependent:
  - local single-machine cities may put them in SQLite;
  - multi-machine or replay/sync-oriented cities should keep them in DoltLite
    or add explicit mirror tables.

Migration target:

- The plugin companion defines profiles:
  - `ledger`: keep issue and workflow state in DoltLite, only local caches in
    SQLite;
  - `local-runtime`: put local workflow/runtime tables in SQLite;
  - `mirror`: local runtime plus explicit DoltLite mirror tables.
- The plugin should eventually provide a real table resolver, not just a
  layout report.
- Gas City should consume the profile/table resolver instead of hardcoding
  placement decisions.

### 6. Raw SQL, VCS, Remotes, And Maintenance

Current behavior:

- Gas City and the pack historically call `bd sql`, `bd flatten`, and `bd gc`
  through DoltLite env.
- The plugin advertises raw SQL support, leases, maintenance, versioning,
  branching, DoltLite remotes, backups, sync, flatten, compact, and GC.
- DoltLite remotes are available through DoltLite SQL functions and
  `doltlite-remotesrv`; the plugin has a focused remote server test harness.

Migration target:

- `bd sql` for DoltLite should be implemented by the plugin backend protocol,
  not copied into Gas City.
- `bd flatten`, `bd gc`, remote operations, backup, and VCS operations should
  be plugin-owned.
- Gas City pack commands can remain as user-facing wrappers, but they should
  invoke `bd`/plugin capabilities instead of embedding DoltLite SQL rules.

### 7. What Should Stay In Gas City

These are not good plugin responsibilities:

- city lifecycle: `gc init`, `gc start`, `gc stop`, supervisor behavior;
- rig discovery, pack installation, and city/rig environment projection;
- formula routing, agent sessions, mayor/workflow orchestration;
- deciding whether a deployment is single-machine local or multi-machine;
- exposing Gas City UX commands and doctor summaries;
- the optional decision to use an in-process fast path for Gas City-specific
  read pressure.

Even if the plugin becomes a GC companion plugin, Gas City remains the
orchestrator. The plugin should own DoltLite storage mechanics and return
machine-readable facts to Gas City.

## Proposed End State

```text
gc init / pack install
  -> installs bd, bd-backend-doltlite, optional gc-doltlite
  -> writes .beads/metadata.json with backend_plugin_command and attached ops DB
  -> validates metadata through gc-doltlite health/layout

bd command
  -> Beads core plugin-process adapter
  -> bd-backend-doltlite serve
  -> plugin DoltLite store

gc hot read path, if still needed
  -> plugin-owned Go package or local process adapter
  -> same table resolver/query semantics as bd-backend-doltlite

gc user-facing maintenance commands
  -> bd/plugin capabilities
  -> DoltLite SQL/remotes/GC/flatten owned by plugin
```

## Migration Order

1. Finish metadata convergence.
   - Make both DoltLite packs write `backend_plugin_command`,
     `backend_plugin_args`, `attached_databases`, and `gascity.doltlite_profile`
     consistently.
   - Use `gc-doltlite health` in doctor/build smoke paths.

2. Move raw SQL and maintenance to the plugin path.
   - Ensure `bd sql` works through the plugin.
   - Keep pack commands as wrappers only.

3. Move the table resolver into the plugin.
   - Promote `internal/gcplugin/layout.go` from reporting to runtime routing.
   - Start with `repo_mtimes` and `local_metadata` in `ops.sqlite`.

4. Collapse duplicate fast-path query semantics.
   - Add parity tests for Gas City claim/ready/list/routed-pool/session/order
     queries against plugin-backed DoltLite.
   - Replace Gas City's fork-local query implementation with a plugin-owned
     library or process-backed adapter.

5. Remove legacy pack wrapper behavior.
   - Stop schema bootstrapping with `sqlite3`.
   - Stop writing divergent metadata shapes.
   - Keep only install, env, smoke, and user command wrapper behavior.

## Review Checklist

Before deleting any current Gas City DoltLite code, prove:

- `gc init` creates city and rig metadata with plugin fields.
- all rigs can run `bd` through the plugin.
- `gc prime`, formula claim queries, ready pool queries, and routed-pool queries
  return the same rows as the current live city.
- `bd sql`, leases, heartbeat/reclaim, remotes, backup, flatten, and GC work
  through the plugin.
- the fast path either still works through a plugin-owned package or is removed
  with measured acceptable performance.
- linked binaries are built against the same `libdoltlite`.
