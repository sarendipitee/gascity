# DoltLite ops.sqlite table audit

This audit asks one question for each Beads/Gas City table:

> Does this data need Dolt/DoltLite version-control semantics: history, diff, merge, clone/pull/push, backup/restore, state hashing, or cross-machine recovery?

If the answer is no, the table is a candidate for an attached SQLite operational database such as `ops.sqlite`.

## Summary

Good first migration candidates:

- `repo_mtimes`
- `local_metadata`
- new diagnostics/metrics/lock-observation tables that do not exist yet

Possible later candidates, but only behind compatibility views or explicit query routing:

- `interactions`
- `wisp_events`
- `wisp_comments`
- closed/expired session operational rows, if Gas City deliberately makes them local-only

Not safe to move as a simple migration:

- `issues`
- `dependencies`
- `labels`
- `comments`
- `events`
- `metadata`
- `config`
- `routes`
- `issue_counter`
- `child_counters`
- `wisps`
- `wisp_labels`
- `wisp_dependencies`
- `custom_statuses`
- `custom_types`
- `federation_peers`
- compaction/history snapshot tables

The strongest immediate result is that `ops.sqlite` should start with tables that upstream Beads already treats as clone-local: `repo_mtimes` and `local_metadata`. Moving workflow tables should wait until we have tests proving `bd ready`, `bd list`, Gas City routed-pool queries, graph roots, reaper SQL, order single-flight, and recovery semantics are unchanged.

## Core issue tables

### `issues`

Version control required.

Columns such as `id`, `title`, `description`, `status`, `priority`, `issue_type`, `assignee`, `metadata`, `created_at`, `updated_at`, `closed_at`, `defer_until`, `due_at`, `lease_expires_at`, `heartbeat_at`, and `row_lock` are all part of durable work state.

Gas City stores agents, sessions, orders, graph roots, messages, nudges, workflow roots, and normal work as issue rows. Even if a particular `issue_type=session` row is operational, the table as a whole participates in ready queries, dependency checks, history, recovery, sync, and reaper cleanup.

Do not move this table wholesale. If we want local-only session state, add a separate `ops.sessions`/`ops.session_runtime` table rather than splitting selected `issues` columns.

### `dependencies`

Version control required.

Dependencies determine readiness, graph topology, parent-child relationships, blockers, and cross-tier links to wisps. The deterministic dependency id work exists specifically because dependency identity must merge correctly across clones.

Do not move to SQLite.

### `labels`

Version control required.

Labels drive selection, routing, order tracking, and user-visible issue state. They must travel with the issue and be diffable/mergeable.

Do not move to SQLite.

### `comments`

Version control required unless a future product decision makes comments local-only.

Comments are part of the user-visible issue record and history. They should sync with the issue.

Do not move to SQLite now.

### `events`

Version control required for current semantics.

Events are audit/history for issue mutation. They can be high-churn, but moving them would change history and recovery behavior.

Do not move to SQLite unless Beads defines events as derived/local cache.

## Configuration and identity tables

### `config`

Version control required.

Global Beads configuration affects behavior and must be available after clone/pull/restore. Upstream has special commit handling for config because it is sensitive during pull, not because it is disposable.

Do not move to SQLite.

### `metadata`

Version control required.

This is durable store metadata. Moving it would make clones/restores incomplete.

Do not move to SQLite.

### `local_metadata`

Version control not required.

Upstream already marks this as clone-local and ignored/nonlocal. It is a strong candidate for `ops.sqlite`.

Migration shape:

1. Create `ops.local_metadata`.
2. Copy existing `local_metadata` rows into `ops.local_metadata`.
3. Replace Beads accessors with a backend-aware table resolver or compatibility view.
4. Leave a compatibility path for older DBs until all callers are routed through the resolver.

### `routes`

Usually version control required.

Routes map prefixes to paths and are part of how a multi-store Beads/Gas City setup resolves scope. Gas City also has `.beads/routes.jsonl`, but the DB table should not be made local-only without proving sync/recovery still reconstructs the same routing graph.

Do not move to SQLite now.

### `issue_counter`

Version control required.

This controls id allocation. If it diverges locally, clones can allocate conflicting ids or skip ranges unpredictably.

Do not move to SQLite.

### `child_counters`

Version control required.

Child id allocation and parent-child naming must remain consistent with durable issue topology.

Do not move to SQLite.

## Snapshots and compaction tables

### `issue_snapshots`

Version control required for current semantics.

Snapshots preserve compacted issue content. Moving them would make a cloned/restored DB unable to reconstruct compacted history.

Do not move to SQLite.

### `compaction_snapshots`

Version control required for current semantics.

Same reasoning as `issue_snapshots`.

Do not move to SQLite.

## Import/export and local state

### `repo_mtimes`

Version control not required.

Upstream already moved this to ignored/local state because independent clones updating mtimes cause merge conflicts. This is the safest first real table to move into `ops.sqlite`.

Migration shape:

1. Create `ops.repo_mtimes`.
2. Copy existing `repo_mtimes`.
3. Route all `repo_mtimes` reads/writes to `ops.repo_mtimes`.
4. Drop or ignore the DoltLite table after compatibility tests pass.

## AI/tooling telemetry

### `interactions`

Probably does not require version control, but needs product confirmation.

This looks like interaction/LLM/tool telemetry. It may be useful locally and high-write. If users expect interaction logs to sync between machines, keep it versioned. If it is diagnostic only, it can move to SQLite.

Candidate after `repo_mtimes` and `local_metadata`.

## Federation/customization tables

### `federation_peers`

Version control required or security-sensitive; do not casually move.

It describes remote peers and may include encrypted credentials. It needs a separate security/design decision, not an ops-cache migration.

Do not move to SQLite now.

### `custom_statuses`

Version control required.

Status vocabulary changes affect how issues are interpreted and must sync with the issue DB.

Do not move to SQLite.

### `custom_types`

Version control required.

Type vocabulary affects issue semantics and must sync.

Do not move to SQLite.

## Wisp tables

### `wisps`

Not safe as a simple SQLite migration.

Upstream marks `wisps` as ignored/nonlocal, but Gas City uses wisps as live workflow state:

- graph roots
- order roots
- routed work
- formula execution topology
- stale workflow root cleanup
- ready/list tier queries
- `gc.root_bead_id` membership
- `gc.root_store_ref`
- order single-flight checks

Columns like `status`, `issue_type`, `assignee`, `metadata`, `created_at`, `updated_at`, `closed_at`, `no_history`, `ephemeral`, `lease_expires_at`, `heartbeat_at`, and `row_lock` are operationally load-bearing.

A future migration might move some wisp rows to SQLite only if we preserve:

- `TierIssues`, `TierWisps`, and `TierBoth` query behavior
- `bd ready` and Gas City ready projection
- routed pool queries
- graph descendant traversal
- reaper cleanup SQL
- cross-store root ownership checks
- claim/heartbeat conflict semantics

Until then, keep `wisps` in the main DoltLite DB.

### `wisp_dependencies`

Not safe as a simple SQLite migration.

This table is graph topology. Gas City traverses it for workflow roots, stale root detection, parent-child ownership, and issue/wisp cross-tier blockers.

Do not move until `wisps` movement is solved.

### `wisp_labels`

Not safe as a simple SQLite migration.

Labels on wisps are used for order tracking and routing. They need to stay with `wisps`.

Do not move until `wisps` movement is solved.

### `wisp_events`

Possible later candidate.

If wisp events are only local audit for ephemeral workflow state, they could move to SQLite. If any recovery, history, or remote debugging depends on them syncing, keep them in DoltLite.

Do not move in the first migration.

### `wisp_comments`

Possible later candidate.

Same logic as `wisp_events`. Likely less dangerous than `wisps` or `wisp_dependencies`, but still coupled to the wisp lifecycle.

Do not move in the first migration.

## Column-level notes

### Agents and sessions

Agents and sessions are represented primarily as `issues` rows plus metadata, not as a separate table. Some session state is durable coordination state and some is local runtime state.

Do not split existing `issues` columns into SQLite. Instead, add explicit `ops` tables for local runtime-only state, for example:

- `ops.session_runtime`
- `ops.agent_observations`
- `ops.provider_processes`
- `ops.transient_claim_attempts`

Closed historical session beads can be pruned; that does not mean active session coordination should be local-only.

### Graph roots

Graph roots need durable version-control semantics today.

Graph roots and descendants are how Gas City recovers workflow topology after crashes, restarts, and cross-store work. They use `issues`/`wisps`, dependencies, labels, and metadata such as `gc.root_bead_id`, `gc.root_store_ref`, `gc.formula_contract`, and `gc.kind`.

Do not move graph roots to SQLite.

### Leases

Lease columns are tempting SQLite candidates because they are high-churn:

- `assignee`
- `lease_expires_at`
- `heartbeat_at`
- `row_lock`

But they are not safe to move column-by-column. `row_lock` exists to force conflicts between heartbeat/close/claim paths. Moving only lease fields out of the main row would remove that conflict behavior unless the claim protocol is redesigned.

A future design could store heartbeat-only observations in SQLite, but the authoritative claim state should remain in the main DB.

### Metadata

Gas City uses `metadata` as the control-plane extension surface. Many keys are durable workflow state:

- `gc.routed_to`
- `gc.root_bead_id`
- `gc.root_store_ref`
- `gc.session_id`
- `gc.session_name`
- `gc.graphv2_root_key`
- branch/refinery handoff fields
- order/workflow outcome fields

Do not move the generic `metadata` JSON column. If a key is local-only, create a typed `ops` table for that purpose instead of splitting JSON.

## Proposed migration order

1. Move `repo_mtimes` to `ops.sqlite`.
2. Move `local_metadata` to `ops.sqlite`.
3. Add new `ops` tables for plugin diagnostics, lock observations, and backend metrics.
4. Evaluate `interactions` with a product decision: synced audit log or local telemetry.
5. Only after conformance tests exist, consider `wisp_events` and `wisp_comments`.
6. Leave `wisps`, `wisp_dependencies`, and `wisp_labels` in DoltLite until Gas City has explicit mixed-main/ops query support and tests for every tier/query/reaper path.

## Tests required before moving anything beyond local metadata

- `bd list` parity across Dolt and DoltLite plugin.
- `bd ready` parity.
- Gas City `.WorkQuery`, `.AssignedInProgressQuery`, `.AssignedReadyQuery`, `.RoutedPoolQuery`, and `.SlingQuery` parity.
- Graph v2 root creation, claim, close, and recovery.
- Order single-flight with `TierBoth`.
- Reaper stale wisp and workflow-root cleanup.
- Lease claim/heartbeat/close race behavior.
- Clone/fetch/pull/restore behavior with `ops.sqlite` present and absent.
