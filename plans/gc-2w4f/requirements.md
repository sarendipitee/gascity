---
schema: gc.build.requirements.v1
workflow:
  id: gc-wisp-kwo
  formula: jj-build
methodology:
  pack: gascity-jj-base
  name: jj-build
producer:
  formula: jj-build
  stage: requirements
  attempt: 1
status: approved
trace:
  upstream:
    - path: beads/gc-wisp-kwo
      hash: bead:gc-wisp-kwo
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
        - REQ-005
        - REQ-006
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
---

# Requirements: DoltLite Init Runtime Isolation

## Problem Statement

DoltLite-linked Gas City installs can inherit dynamic loader variables from the
operator shell or from a long-running tmux server. When `LD_LIBRARY_PATH` or
`DYLD_LIBRARY_PATH` points at a local development tree, child `gc` or `bd`
launchers can bind to the wrong `libdoltlite` even after the beads-doltlite pack
has installed a selected binary and library pair.

Fresh DoltLite city initialization also needs to avoid claiming closed routed
work, preserve the blocked state reported by `bd`, and use the pack version that
contains the intended reproducible init behavior. The result should make fresh
DoltLite-backed cities deterministic without changing non-DoltLite city startup
behavior.

## W6H

| Question | Answer |
| --- | --- |
| Who | Operators and workflow agents running fresh Gas City installs with the DoltLite beads backend. |
| What | Scrub ambient dynamic loader paths when launching DoltLite readiness checks and DoltLite-backed agent sessions, while preserving correct routed-work state handling. |
| When | During `gc init` provider readiness checks, session startup for DoltLite-backed cities, and hook claiming of routed workflow work. |
| Where | The Gas City CLI runtime paths under `cmd/gc`, the bd-backed task store adapter under `internal/beads`, and the public beads-doltlite pack pin. |
| Why | To prevent stale local libraries or stale routed work from making a fresh DoltLite city unreliable. |
| How | Centralize DoltLite loader environment scrubbing, apply it only to DoltLite-backed launch paths, map `blocked` as a durable status, and require hook candidates to be open before claiming. |
| How much | Cover the local DoltLite init and workflow routing behavior needed by this integration branch; do not redesign all runtime environment handling. |

## User Stories

### REQ-001: Scrub loader paths for DoltLite readiness checks

As an operator initializing a DoltLite-backed city, I need provider readiness
checks to ignore stale dynamic loader variables so `gc` and `bd` validate
against the installed DoltLite library.

Acceptance criteria:

- Readiness checks clear `LD_LIBRARY_PATH` and `DYLD_LIBRARY_PATH` for discovered
  `gc` and `bd` command probes.
- Existing `GC_DOLTLITE_SKIP_LOCAL_LIB` and `GC_DOLTLITE_SKIP_LOCAL_SOURCE`
  safeguards remain present during readiness probes.
- Tests cover inherited loader variables and assert that the child probe sees
  cleared values.

### REQ-002: Scrub loader paths for DoltLite-backed agent sessions

As a workflow operator, I need agent sessions in a DoltLite-backed city to launch
without inheriting stale loader paths from the parent environment or tmux server.

Acceptance criteria:

- Session startup adds empty `LD_LIBRARY_PATH` and `DYLD_LIBRARY_PATH` values
  when the configured beads backend resolves to `doltlite`.
- Non-DoltLite cities do not receive DoltLite-specific loader scrubbing.
- Tests verify the prepared session environment contains explicit empty loader
  values for DoltLite-backed city configs.

### REQ-003: Keep DoltLite backend detection centralized

As a maintainer, I need DoltLite-specific runtime behavior to use one small
backend detection helper so future launch paths do not duplicate backend-name
logic.

Acceptance criteria:

- DoltLite loader scrub values are produced by a shared CLI helper.
- Backend detection safely handles nil city config.
- Backend detection uses the existing beads backend resolver rather than string
  matching raw config fields at every call site.

### REQ-004: Preserve blocked bead status in Gas City projections

As a workflow worker, I need blocked beads to remain visibly blocked even when
older `bd list` rows do not include an `is_blocked` flag.

Acceptance criteria:

- The bd store maps `blocked` to a distinct Gas City bead status.
- Assigned-work filtering treats either `status=blocked` or `is_blocked=true`
  as known blocked work.
- Tests cover a blocked list row whose `is_blocked` value is absent.

### REQ-005: Do not claim closed routed work

As a routed worker, I need `gc hook --claim` to ignore closed unassigned work so
completed beads cannot be re-entered by a later session.

Acceptance criteria:

- Hook claim candidates must have a non-empty ID, no assignee, a matching route,
  and `status=open`.
- Tests cover a closed routed unassigned bead and expect a drain/no-work result.
- Existing route matching behavior remains unchanged for open candidates.

### REQ-006: Pin fresh DoltLite init to the intended pack revision

As a maintainer, I need fresh DoltLite init output to use the beads-doltlite pack
revision that contains reproducible init source and library selection fixes.

Acceptance criteria:

- The public beads-doltlite pack version constant points at the intended commit
  SHA.
- The update stays isolated to the public pack pin and does not change unrelated
  pack sources.
- The requirements artifact records this pin as part of the DoltLite init
  reliability scope.

## Technical Stories

### TS-001: Central helper for loader scrubbing

Add a CLI-local helper that returns explicit empty `LD_LIBRARY_PATH` and
`DYLD_LIBRARY_PATH` assignments and a helper that identifies DoltLite-backed city
configs through the existing backend resolver.

### TS-002: Readiness and session launch integration

Merge the scrubbed loader environment into DoltLite readiness probes and into
prepared session environment values only when the city uses the DoltLite backend.

### TS-003: Routed work state hardening

Keep `blocked` distinct when mapping bd statuses and require open status before
`gc hook --claim` treats an unassigned routed bead as claimable.

## Behavior Requirements

- DoltLite-specific environment scrubbing must be explicit and deterministic for
  child launchers.
- Non-DoltLite cities must not get DoltLite-only loader environment behavior.
- Hook claiming must not re-open or re-enter closed work.
- Blocked work must remain distinguishable from open work in Gas City's bead
  projections and pool-demand filters.
- The pack pin must remain a single-source constant for fresh DoltLite init.

## Example Mapping

| Requirement | Implementation surface |
| --- | --- |
| REQ-001 | `cmd/gc/init_provider_readiness.go` and readiness tests |
| REQ-002 | `cmd/gc/session_lifecycle_parallel.go` and session lifecycle tests |
| REQ-003 | `cmd/gc/doltlite_loader_env.go` |
| REQ-004 | `internal/beads/bdstore.go`, `cmd/gc/assigned_work_scope.go`, and related tests |
| REQ-005 | `cmd/gc/cmd_hook_claim.go` and hook claim tests |
| REQ-006 | `internal/config/public_packs.go` |

## Acceptance Criteria

- Focused Go tests cover loader scrubbing for readiness probes and session
  launch preparation.
- Focused Go tests cover blocked status projection and assigned-work filtering.
- Focused Go tests cover closed routed work being skipped by `gc hook --claim`.
- The requirements document is recorded in the workflow manifest with schema,
  SHA-256 content hash, and jj change ID.

## Out Of Scope

- Redesigning all runtime environment handling for every backend.
- Changing DoltLite storage semantics or the bd CLI wire format.
- Reworking graph.v2 routing, workflow scheduling, or session affinity.
- Introducing role-specific logic into the SDK.

## Open Questions

- None.
