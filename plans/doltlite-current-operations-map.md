# Current DoltLite operations map

This maps the DoltLite operations currently present in Gas City and the DoltLite backend plugin.

There are three layers:

1. Gas City metadata/bootstrap wiring.
2. Gas City's optimized DoltLite fast path.
3. The backend plugin's full Beads-compatible DoltLite store.

## 1. Gas City metadata/bootstrap wiring

### `.beads/metadata.json`

Written by:

- `cmd/gc/beads_provider_lifecycle.go`
- `internal/bootstrap/packs/beadsdoltliteinit/assets/scripts/gc-beads-doltlite-bd.sh`
- legacy/full `examples/bd/assets/scripts/gc-beads-bd.sh`

Fields currently used for DoltLite:

- `backend = "doltlite"`
- `database = "doltlite"`
- `dolt_database`
- `project_id`
- `attached_databases`
  - `alias = "ops"`
  - `path = "<scope>/.gc/ops.sqlite"`

This is not a table operation, but it controls which DB is opened and which attached DBs are visible.

### Attached DB handling

`internal/beads/doltlite_read_store.go` reads `.beads/metadata.json`, then executes:

```sql
ATTACH DATABASE ? AS <alias>
```

Current attached alias:

- `ops`

Current use:

- Attached, but no Beads/Gas City tables are currently routed to `ops.*`.
- Tests create/open `ops.sqlite`, but the live table logic still reads and writes main tables.

## 2. Gas City optimized DoltLite fast path

Primary file:

- `internal/beads/doltlite_read_store.go`
- `internal/beads/doltlite_count.go`

This layer is a Gas City store implementation optimized for local DoltLite reads/writes. It does not implement every Beads backend operation; it handles the paths Gas City uses heavily.

### Tables read for list/count/ready

#### `issues`

Read columns:

- `id`
- `title`
- `status`
- `issue_type`
- `priority`
- `created_at`
- `updated_at`
- `assignee`
- `description`
- `metadata`
- `ephemeral`
- `no_history`

Operations:

- `SELECT COUNT(*)`
- filtered `SELECT` for `List`
- dedupe against durable wisp rows
- ready work stays on `issues` for raw ready unless wisps are explicitly requested

#### `wisps`

Read columns:

- same shape as `issues`
- `ephemeral`
- `no_history`
- `is_blocked` when present

Operations:

- filtered `SELECT` for `TierWisps`
- merged with `issues` for `TierBoth`
- contributes non-ephemeral/no-history rows to `TierIssues`
- counted in the fast-path hash/fingerprint

#### `labels` / `wisp_labels`

Read columns:

- `issue_id`
- `label`

Operations:

- label filtering via `EXISTS`
- label hydration after list/show
- active label aggregation

#### `dependencies` / `wisp_dependencies`

Read columns:

- `issue_id`
- `depends_on_id` or typed target columns:
  - `depends_on_issue_id`
  - `depends_on_wisp_id`
  - `depends_on_external`
- `type`

Operations:

- ready/blocker predicates
- parent-child relationships
- dependency hydration
- reverse dependency lookup
- mixed issue/wisp blocker joins

### Tables written by Gas City fast path

#### `wisps`

Writes:

- `INSERT` new wisp rows
- `UPDATE metadata`
- `UPDATE status`
- `UPDATE` general fields through `Update`
- `DELETE` by id

Columns inserted on direct wisp create:

- `id`
- `title`
- `status`
- `issue_type`
- `priority`
- `created_at`
- `updated_at`
- `assignee`
- `description`
- `design`
- `acceptance_criteria`
- `notes`
- `metadata`
- `ephemeral` if column exists
- `no_history` if column exists

#### `wisp_labels`

Writes:

- `INSERT`
- `DELETE`

Columns:

- `issue_id`
- `label`

#### `wisp_dependencies`

Writes:

- `SELECT COUNT` to preserve existing `parent-child` edges
- `DELETE` non-parent duplicate dependency
- `INSERT` dependency
- `DELETE` dependency

Columns:

- `id`
- `issue_id`
- one of:
  - `depends_on_issue_id`
  - `depends_on_wisp_id`
  - `depends_on_external`
  - legacy `depends_on_id`
- `type`
- `created_by`
- `metadata`

#### `issues`

Writes:

- `UPDATE metadata`
- `UPDATE status`
- `UPDATE` general fields through `Update`
- `DELETE` by id

The fast path does not usually create durable `issues`; normal creation goes through the backend/store path.

#### `labels` / `dependencies`

Writes happen when the target bead is in the durable issue tier:

- label add/remove
- dependency add/remove
- parent-child replacement
- delete cascade cleanup

### Fast-path cache/fingerprint

`currentDoltHash` is not a real Dolt commit hash. It fingerprints table counts and max timestamps/counts:

- `issues`: count + max `updated_at`
- `wisps`: count + max `updated_at`
- `labels`: count
- `wisp_labels`: count
- `dependencies`: count
- `wisp_dependencies`: count

This means moving any of these tables to `ops.sqlite` requires updating the fingerprint to use the qualified table location.

## 3. Backend plugin DoltLite store

Primary repo:

- `/data/projects/doltlite-gascity/rigs/beads-backend-doltlite-plugin`

Primary files:

- `internal/storage/doltlite/*`
- `internal/storage/issueops/*`
- `internal/storage/sqlbuild/*`
- `cmd/bd-backend-doltlite/main.go`

This is the full backend. Most normal Beads operations route through shared `issueops` helpers.

### Core CRUD and workflow tables

#### `issues`

Operations:

- `INSERT`/upsert on create/import
- `SELECT` by id/list/search/ready
- `UPDATE` status, assignee, metadata, priority, title, body fields, dates, lease fields
- `DELETE`

Important columns:

- identity/content: `id`, `title`, `description`, `design`, `acceptance_criteria`, `notes`
- lifecycle: `status`, `created_at`, `updated_at`, `closed_at`, `closed_by_session`, `close_reason`
- scheduling: `priority`, `issue_type`, `assignee`, `defer_until`, `due_at`, `started_at`
- tiering: `ephemeral`, `no_history`
- control-plane: `metadata`
- blocking: `is_blocked`
- lease/recovery: `lease_expires_at`, `heartbeat_at`, `row_lock`

#### `wisps`

Operations mirror `issues`, but route ephemeral/no-history/transient rows.

Important columns are mostly the same as `issues`.

Important difference:

- lease columns exist for shared SQL shape, but reclaim only targets permanent `issues`; comments say wisps are ephemeral and not reclaimed.

#### `labels` / `wisp_labels`

Operations:

- `SELECT label`
- batched label hydration
- `INSERT IGNORE`
- `DELETE`

Columns:

- `issue_id`
- `label`

#### `dependencies` / `wisp_dependencies`

Operations:

- `INSERT`
- `UPDATE metadata`
- `DELETE`
- dependency cycle checks
- ready/blocker queries
- parent-child traversal
- source/target retargeting for rename/promote
- mixed issue/wisp dependency resolution

Important columns:

- `id`
- `issue_id`
- `depends_on_issue_id`
- `depends_on_wisp_id`
- `depends_on_external`
- generated/legacy `depends_on_id`
- `type`
- `created_at`
- `created_by`
- `metadata`
- `thread_id`

#### `events` / `wisp_events`

Operations:

- `INSERT` mutation events
- `SELECT` events for an issue
- `SELECT` events since timestamp across both event tables
- `DELETE` during cascade cleanup

Columns:

- `id`
- `issue_id`
- `event_type`
- `actor`
- `old_value`
- `new_value`
- `comment`
- `created_at`

#### `comments` / `wisp_comments`

Operations:

- `INSERT`
- `SELECT`
- `DELETE` during cascade cleanup

Columns:

- `id`
- `issue_id`
- `author`
- `text`
- `created_at`

### Config and local metadata

#### `config`

Operations:

- `SELECT value`
- `INSERT` / `REPLACE` through config helpers
- commit/pull special handling

Important keys:

- custom types/statuses
- issue id mode/prefix
- `kv.*` memory/config rows

This table is currently versioned/durable.

#### `metadata`

Operations:

- store-level metadata key/value reads and writes
- federation last-sync bookkeeping

This table is currently durable.

#### `local_metadata`

Operations:

- key/value local metadata reads and writes

Upstream already treats this as clone-local / ignored / nonlocal.

Candidate for `ops.local_metadata`.

#### `repo_mtimes`

Operations:

- local import/export mtime bookkeeping

Upstream already treats this as clone-local / ignored / nonlocal.

Candidate for `ops.repo_mtimes`.

### Counters and schema vocabulary

#### `issue_counter`

Used for counter-mode ids.

Keep in main unless id allocation is redesigned.

#### `child_counters`

Used for child id allocation/reconciliation.

Keep in main unless child id allocation is redesigned.

#### `custom_statuses` / `custom_types`

Used to interpret issue rows.

Keep in main.

#### `federation_peers`

Used for federation remote config.

Keep in main unless federation config becomes local-only by design.

## DoltLite VCS/system operations

The plugin exposes and uses DoltLite VCS functions directly.

### Commit/status/hash

Operations:

- `SELECT dolt_add(?)`
- `SELECT dolt_commit('-m', ?, '--author', ?)`
- `SELECT dolt_commit('-A', '-m', ?, '--author', ?)`
- `SELECT table_name FROM dolt_status`
- `SELECT dolt_hashof('HEAD')`

Used by:

- normal commit
- commit with config
- commit pending
- batch commit messages
- current commit/hash reporting

### History/diff/as-of

Operations:

- `dolt_diff_issues('HEAD', 'WORKING')`
- `dolt_at_issues(?)`
- `dolt_diff_issues(?, ?)`
- `dolt_history_issues`
- `SELECT commit_hash, committer, email, date, message FROM dolt_log`

Affected table:

- currently `issues` only for issue history/diff/as-of helpers.

This is a strong reason durable issue rows should stay in main.

### Branch/checkout/merge/conflicts

Operations:

- `SELECT dolt_branch(?)`
- `SELECT dolt_checkout(?)`
- `SELECT dolt_branch('-D', ?)`
- `SELECT dolt_merge(?)`
- `SELECT dolt_conflicts_resolve(?, ?)`

Used by:

- branch creation
- checkout
- delete branch
- merge
- conflict resolution
- flatten/compact workflows

### Remotes/sync/backup

Operations:

- `SELECT url FROM dolt_remotes WHERE name = ?`
- `SELECT count(*) FROM dolt_remotes WHERE name = ?`
- `SELECT dolt_remote('remove', ?)`
- `SELECT dolt_remote('add', ?, ?)`
- `SELECT dolt_push(?, ?)`
- `SELECT dolt_push(?, ?, '--force')`
- `SELECT dolt_pull(?, ?)`
- `SELECT dolt_fetch(?, ?)`
- backup implemented as remote add/push/fetch/reset

### Maintenance

Operations:

- `SELECT dolt_gc()`
- `SELECT dolt_log`
- `SELECT dolt_reset(...)`
- `SELECT dolt_cherry_pick(?)`
- flatten/compact branch workflows

## Placement implications

### Easy `ops.sqlite` candidates

- `repo_mtimes`
- `local_metadata`

They are already local/nonlocal in upstream Beads and have little coupling to ready/claim/history.

### Plausible `ops.sqlite` candidates, with resolver work

- `wisps`
- `wisp_labels`
- `wisp_dependencies`
- `wisp_events`
- `wisp_comments`

Why plausible:

- upstream already marks `wisps` and `wisp_*` as ignored/nonlocal
- most operations route through table-name structs or hardcoded paired tables
- attached DB can still be joined/unioned from the same connection

What must change:

- table name resolver must return `ops.wisps` etc.
- schema creation/migrations must create those tables in `ops`
- fast-path fingerprint must count qualified tables
- ready/list/count/dep queries must use qualified names everywhere
- reaper and Gas City direct SQL must use the same resolver or views

### Not good `ops.sqlite` candidates

- `issues`
- `dependencies`
- `labels`
- `comments`
- `events`
- `config`
- `metadata`
- `issue_counter`
- `child_counters`
- `custom_statuses`
- `custom_types`
- `federation_peers`

These are tied to durable identity, history/diff/as-of, id allocation, or sync/merge semantics.

### Column-level split to avoid

Do not split lease columns out of `issues`/`wisps`:

- `assignee`
- `status`
- `lease_expires_at`
- `heartbeat_at`
- `row_lock`

The lease design depends on mutating the same row and especially `row_lock` to force Dolt serialization conflicts.
