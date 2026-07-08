---
schema: gc.build.implementation-summary.v1
workflow:
  id: gc-jjpc
  formula: jj-do-work
methodology:
  pack: gascity-jj-base
  name: jj-do-work
producer:
  formula: jj-do-work
  stage: implement
  attempt: 1
status: approved
trace:
  upstream:
    - path: beads/gc-vgvj
      hash: bead:gc-vgvj
      ids:
        - REQ-001
        - REQ-002
        - REQ-003
        - REQ-004
        - REQ-005
        - REQ-006
        - REQ-007
    - path: requirements.md
      hash: sha256:3acc0e95ba45ba99a38b66ee92f358e6e0b8959e4214f9e29e0047ee1833e763
    - path: plan.md
      hash: sha256:6f0c6911a55ed6f079b60fd1837747bca4beefe89b34fbd917766956a9a315fd
    - path: decomposition.md
      hash: sha256:e245c72107094d6c4d035bd2e95ff1abb3b66585bea979ac03aa188f42980fe2
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
---

## Summary

Produced a manifest-managed, document-only audit handoff for the controller demand vs upstream investigation. No source files were changed.

| ID | Status |
| --- | --- |
| REQ-001 | covered |
| REQ-002 | covered |
| REQ-003 | covered |
| REQ-004 | covered |
| REQ-005 | covered |
| REQ-006 | covered |
| REQ-007 | covered |

## Intended Behavior

Direct work discovery and controller demand should agree about normal worker-routed ready work. If `gc hook --claim --json` can claim a bead routed to `gascity/gc.implementation-worker`, the controller demand path should have a regression guard that proves the normal worker route appears in desired-state demand or `scale_check_counts` when the pool is cold and has no custom `scale_check`.

## Changed Files

- `plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/implementation-summary-gc-vgvj.md`: new audit handoff and implementation summary.
- `plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/manifest.json`: records the implementation-summary document path, schema, hash, source workspace identity, source change ID, and document change ID.

Source workspace: `/data/projects/doltlite-gascity/gascity/.gc/workspaces/gascity/packs/gascity-jj-base`

Source change ID: `yqwwsuuurskrqytnovpqzstsmyywnmml`

Document workspace: `/data/projects/doltlite-gascity/gascity`

Document path: `/data/projects/doltlite-gascity/gascity/plans/fork-upstream-audit-20260624/audit-controller-demand-vs-upstream/implementation-summary-gc-vgvj.md`

Document change ID: `kypwosprvvuqyurwlnrmlysyzrptnmvk`

## Verification

| Command | Result |
| --- | --- |
| `jj -R /data/projects/doltlite-gascity/gascity/.gc/workspaces/gascity/packs/gascity-jj-base st` | pass: source workspace was clean before audit collection. |
| `jj -R /data/projects/doltlite-gascity/gascity/.gc/workspaces/gascity/packs/gascity-jj-base log -r @ --no-graph` | pass: source `@` is described as `audit: compare controller demand against upstream`. |
| `gc trace show --since 30m --json` | pass: recent trace records contained live demand evidence for the audit. |
| `jj diff --git --from main --to @ -- cmd/gc/build_desired_state.go cmd/gc/city_runtime.go` | pass: demand code comparison collected. |
| `jj diff --git --from main --to @ -- cmd/gc/scale_from_zero_no_scalecheck_test.go cmd/gc/scale_from_zero_named_no_scalecheck_test.go internal/beads/caching_store_test.go` | pass: focused test comparison collected. |
| `jj diff --git --from main --to @ -- internal/beads/doltlite_read_store.go internal/beads/bdstore.go internal/beads/doltlite_count.go internal/beads/doltlite_read_store_test.go internal/beads/caching_store_test.go` | pass: DoltLite/read-path comparison collected. |
| `GC_BEAD_ID=gc-vgvj /data/projects/doltlite-gascity/gascity-packs/gascity/assets/scripts/checks/build-artifact-valid.sh` | pass: validated this implementation summary artifact. |

## Audit Handoff

### Baselines

- `main`: change `nqstmomxqroksplxqktusomlkpsrxvwo`, commit `41c54dcddc24`, `Persist Dolt server mode in canonical beads config (#3719)`.
- `pr/runtime-ready-demand-snapshot`: local change `nwyvrryoxzzmpmwozmkmuuxmvsxuvpqn`, commit `bda1fd03c15a`, `fix(runtime): refresh demand snapshots for routed work`.
- `pr/runtime-ready-demand-snapshot@origin`: divergent change `nwyvrryo/4`, commit `4a6be657`, same subject.
- Source workspace `@`: change `yqwwsuuurskrqytnovpqzstsmyywnmml`, commit `8e52b09e`, `audit: compare controller demand against upstream`.

Commands used:

```bash
jj log -r main --no-graph
jj log -r pr/runtime-ready-demand-snapshot --no-graph
jj bookmark list
```

### Symptom Evidence

Direct discovery evidence came from this step's routed claim path. `gc hook --claim --json` claimed `gc-vgvj`, and `bd show gc-vgvj --json` records:

- `gc.routed_to=gascity/gc.implementation-worker`
- `assignee=gc__implementation-worker-dg-wisp-36dgs0`
- `status=in_progress`

Recent controller trace evidence came from:

```bash
gc trace status
gc trace show --since 30m --json
```

Trace record `cycle-2acf4d9a34055506:396419:33` at `2026-06-25T09:20:42.875390477Z` shows `pool_desired` includes `gascity/gc.implementation-worker: 2`, while `scale_check_counts` contains only:

- `beads-doltlite/core.control-dispatcher: 1`
- `gascity-dashboard/core.control-dispatcher: 1`
- `gascity/core.control-dispatcher: 1`
- `lightjj/core.control-dispatcher: 1`

That reproduces the documented mismatch class: normal worker-routed work is visible to direct discovery, but the controller trace's `scale_check_counts` surface is limited to control-dispatcher routes.

### Upstream And Local Demand Findings

The local source change differs from `main` in `cmd/gc/build_desired_state.go` and `cmd/gc/city_runtime.go`.

Confirmed facts:

- `readyForControllerDemandQuery` sets `ReadyQuery.TierMode = beads.TierBoth`.
- It reads `handles.Cached.Ready(query)` first, falls back to `handles.Live.Ready(query)` when the cache is unavailable, and, for explicit store handles, performs a live `Ready` freshness read even when cached rows exist.
- Normal default scale-check demand groups ready rows by backing store and increments `counts[template]` when an unassigned ready bead's `gc.routed_to` or legacy workflow route candidate matches `group.templates`.
- The local branch adds named-session route support through `defaultScaleCheckTarget.namedIdentities` and `addNamedDemandRoute`, so identity-routed demand can set `NamedSessionDemand` instead of only waking a backing pool.
- The local branch adds `readyDemandFingerprint` to runtime demand snapshots and refreshes the snapshot when ready-demand identity changes.

Hypotheses:

- The live `scale_check_counts` omission is most likely a coverage gap around the normal worker default scale-check path, not a proven DoltLite native-read bug.
- The named-session fix path does not by itself prove normal worker pool routes are counted, because the existing named test expects identity-routed demand to set `NamedSessionDemand` and keep pool `ScaleCheckCounts` at zero.

### DoltLite And Cache Read-Path Classification

The controller demand path calls the Go `beads.Store` interface, not the `bd ready` CLI directly. In this branch it uses cached ready rows plus a live ready read when store handles expose both cache and backing store.

Existing cache tests already cover two relevant stale-row classes:

- `TestCachingStoreCachedReadyDeclinesAfterDroppedRoutingEvent` expects `CachedReady()` to decline and force authoritative `ReadyLive` fallback after a stale ready-candidate event path.
- `TestCachingStoreCachedReadyReflectsRoutedWorkReleaseAfterSessionClose` covers released routed work becoming visible to the demand path after session close.

The DoltLite diff against `main` is broad and includes write locks, conditional status update fallback, and native/CLI read-path work. Current evidence does not prove that `internal/beads/doltlite_read_store.go`, `internal/beads/bdstore.go`, or `internal/beads/doltlite_count.go` caused the controller-demand mismatch. Treat DoltLite as a risk to keep covered, not the first regression target.

### Focused Regression Target

Add or extend a `cmd/gc` demand test before changing runtime behavior. The strongest target is `cmd/gc/scale_from_zero_no_scalecheck_test.go`, with an assertion that a cold normal worker route visible through a ready bead appears in desired-state demand or `scale_check_counts` for the same template the claim path uses.

The test should model a ready, unassigned bead with:

```go
Metadata: map[string]string{"gc.routed_to": "gascity/gc.implementation-worker"}
```

Expected assertion shape:

- `ScaleCheckCounts["gascity/gc.implementation-worker"]` is non-zero when the normal worker pool is cold and has no custom `scale_check`, or
- the desired-state surface that replaces `scale_check_counts` explicitly contains that route.

Do not rely only on `core.control-dispatcher` routes or named-session identity demand tests; those can pass while normal worker route demand remains absent from `scale_check_counts`.

### Unanswered Questions And Follow-Up

- Confirm whether `scale_check_counts` is intended to include all normal worker demand, or whether a newer desired-state surface should be the canonical assertion target.
- If the normal worker demand test passes locally, investigate why the live trace omitted `gascity/gc.implementation-worker` from `scale_check_counts` despite `pool_desired` including it.
- If a normal worker demand test fails only under DoltLite-backed stores, add a second focused cache/native-read test before changing `internal/beads`.

## Remaining Risks

The default document workspace already had unrelated managed-document changes before this bead wrote its summary and manifest entry. This step did not revert or rewrite those changes. Runtime source changes and follow-up beads remain out of scope for this document-only audit handoff.
