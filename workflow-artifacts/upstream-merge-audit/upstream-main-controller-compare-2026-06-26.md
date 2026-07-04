# Upstream Main Controller Comparison - 2026-06-26

Upstream workspace:

- `/data/projects/doltlite-gascity/gascity-upstream-main`
- `main@upstream`: `273f6c3ab41f test: Isolate tests from host GC_* / DOLT_* environment leakage (#3747)`

Current workspace:

- `/data/projects/doltlite-gascity/gascity`
- `@`: `be82635e Document upstream merge audit artifacts`
- parent merge: `6ac59465 Merge upstream main into DoltLite integration line`

## Same-session continuation

The shared same-session continuation path is the same in upstream main and the current workspace:

- `internal/graphroute/graphroute.go`
- `internal/dispatch/drain.go`
- `cmd/gc/cmd_sling.go`
- `cmd/gc/assigned_work_scope.go`
- `cmd/gc/session_resolve.go`
- `cmd/gc/session_template_start.go`
- `cmd/gc/named_sessions.go`

`cmd/gc/cmd_hook_claim.go` differs only around JSON metadata decoding in the current workspace. The continuation preassignment entry point is otherwise the same.

Important backend caveat: "same" here only means the shared controller/continuation source files are the same. It does not mean a DoltLite-backed city has identical runtime behavior to upstream's default store path. The controller still consumes `beads.Store` results, and DoltLite has backend-specific read behavior in `internal/beads/doltlite_read_store.go` plus bd-store bridge behavior.

Conclusion: upstream main does not contain a newer shared same-session/graph-route fix that the current workspace lacks. Backend-specific DoltLite parity still needs to be checked separately.

## Named-session demand

The current workspace is ahead of upstream main for identity-routed named-session demand.

Upstream `defaultNamedSessionDemand` only preserves partial-query retention and does not infer named-session demand from `gc.routed_to` or `gc.run_target`. Its comment says named sessions wake from direct `Assignee=<identity>` work.

The current workspace extends named demand so ready unassigned work whose route matches a configured named identity can materialize the named session. It also tracks backing named identities so distinct backing-template routes do not create duplicate pool phantoms.

Conclusion: if the issue is "named session did not start for identity-routed work", upstream main is behind the current workspace. The fix area is our local named-demand logic and pack configuration, not an upstream-only patch.

## Pack workspace path

Upstream main derives pack-routed work directories with:

- `packDir := filepath.Join(filepath.Dir(base), pack)`

The current workspace adds:

- `Agent.Pack`
- `Agent.PackRoot`
- `poolTriggerPackRoot`
- `gc.pack_root` session metadata
- `GC_PACK_ROOT`

Conclusion: upstream main does not have the `PackRoot` override support. If a source workspace resolves under `.../.gc/workspaces/lightjj/packs/lightjj`, upstream main is not the fix. The current workspace has the hook needed to override the root, but the imported pack config must actually set `pack_root` correctly.

## Bead visibility / Ready work

Upstream main includes `convoy` in `internal/beads/beads.go` `readyExcludeTypes`.

The current workspace does not. Its exclusion map includes `merge-request`, `gate`, `molecule`, `step`, `message`, `session`, `agent`, `role`, and `rig`, but not `convoy`.

This matters for DoltLite too: `internal/beads/doltlite_read_store.go` builds its Ready type predicate from the shared `readyExcludeTypes` map.

The current workspace also differs from upstream in DoltLite Ready SQL. The current workspace has additional DoltLite parity work around multiple assignees, tier mode, failed dependencies, and dependency target expressions. Upstream main still has comments describing a known DoltLite Ready parity gap in that file.

Conclusion: this is a concrete missing upstream fix and a backend-sensitive area. Without the convoy exclusion, controller demand/scale checks can treat convoy container beads as ready work, which can produce workers that start, find no actionable bead, drain, and then repeat.

## Rig-store autoclose

Upstream main resolves the owning bead store across the city and rigs before autoclosing:

- `cmd/gc/cmd_convoy.go`
- `cmd/gc/molecule_autoclose.go`
- `cmd/gc/wisp_autoclose.go`

The current workspace still opens the cwd/store-root store directly in those autoclose entry points.

Conclusion: this is another concrete missing upstream fix. It can leave rig-store convoys, molecules, or wisps open when the `bd on_close` hook runs from the city supervisor context, which can make formula workflows look stuck.

## Provider lifecycle timeout

Upstream main gives provider operation `init` the long timeout:

- `start`, `recover`, and `init`: 120s
- everything else: 30s

The current workspace only gives `start` and `recover` the long timeout; `init` still gets 30s.

Conclusion: this is a concrete missing upstream fix for slow rig bead-store initialization. It can contribute to startup/reconciler churn in Dolt-backed rigs.

## Dolt cleanup

Upstream main skips rigs whose bead metadata declares a non-Dolt backend before building cleanup protections.

The current workspace treats missing `dolt_database` metadata as a force blocker even for non-Dolt-backed rigs.

Conclusion: this is a missing upstream cleanup fix. It is probably not the same-session/named-session root cause, but it matters for DoltLite/Dolt maintenance reliability.

## Test environment isolation

Upstream main includes test-only changes to isolate tests from host `GC_*` and `DOLT_*` environment leakage.

Conclusion: this does not change controller behavior directly, but we should carry it over so DoltLite-related tests do not pass or fail depending on the operator shell.

## Practical merge targets

Carry over these upstream fixes carefully:

1. Exclude `convoy` from Ready work.
2. Resolve owning stores for convoy, molecule, and wisp autoclose.
3. Give provider `init` the long lifecycle timeout.
4. Skip non-Dolt-backed rigs in `gc dolt cleanup`.
5. Add the upstream test environment isolation.

Keep and verify the current workspace changes for:

1. identity-routed named-session demand,
2. `PackRoot` workdir routing,
3. hook-claim JSON metadata decoding.
