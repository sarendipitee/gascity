---
schema: gc.build.plan.v1
workflow:
  id: gc-09rm
  formula: jj-build
methodology:
  pack: gascity-jj-base
  name: jj-build
producer:
  formula: jj-build
  stage: plan
  attempt: 1
plan_slug: audit-doltlite-ready-parity
phase: implementation-plan
rig: gascity
rig_root: /data/projects/doltlite-gascity/gascity
artifact_root: /data/projects/doltlite-gascity/gascity/plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity
requirements_file: /data/projects/doltlite-gascity/gascity/plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity/requirements.md
status: draft
created_at: 2026-06-25T08:28:03Z
updated_at: 2026-06-25T08:28:03Z
trace:
  upstream:
    - path: plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity/requirements.md
      hash: sha256:4150a0e77c0916cdde69931b80339c77da88c11076a2ccdda05ecf6ca78c5e97
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

# Implementation Plan: Audit DoltLite Ready Parity

## Summary

Produce an evidence-based readiness audit for the fork's DoltLite-backed beads
integration. The implementation work is an audit and reporting pass, not a
source-fix pass: downstream workers should inspect current code, tests, docs,
and selected history; write a findings artifact under the existing default@
artifact root; and report follow-up gaps as explicit work instead of patching
production code opportunistically.

The durable document handoff stays in
`plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity/manifest.json`.
Live workflow state remains in the DoltLite-backed bead store. The audit report
should be another manifest-managed document in the same artifact root.

## Current System

The current branch has several distinct surfaces that need to be audited
together.

- Workflow documents live in the `default@` jj workspace under
  `plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity/`.
  `manifest.json` already records `requirements.md` with schema
  `gc.build.requirements.v1`, a SHA-256 hash, and jj change ID.
- Runtime work lives in beads. The requirements explicitly separate the live
  DoltLite-backed bead store from checked-in planning artifacts, so no document
  body should be copied into bead metadata.
- Provider selection and session/provider routing are centered in
  `cmd/gc/providers.go`, with exec-store targeting and raw-provider handling
  covered by `cmd/gc/store_target_exec_test.go` and `cmd/gc/providers_test.go`.
- Managed Dolt/DoltLite lifecycle, port publication, canonical scope files,
  stale export cleanup, and lifecycle serialization are in
  `cmd/gc/beads_provider_lifecycle.go`, `cmd/gc/bd_env.go`,
  `cmd/gc/dolt_runtime_publication.go`, `cmd/gc/dolt_start_managed_test.go`,
  `cmd/gc/dolt_lifecycle_race_test.go`, and `cmd/gc/cmd_stop_test.go`.
- Store behavior is split across `internal/beads/bdstore.go`,
  `internal/beads/native_dolt_store.go`, `internal/beads/doltlite_read_store.go`,
  `internal/beads/caching_store*.go`, `internal/beads/exec/`, and
  `internal/beads/contract/`.
- Hook and ready-query behavior spans `cmd/gc/hook_cross_store.go`,
  `cmd/gc/claim_cross_store_test.go`, `cmd/gc/build_desired_state_test.go`,
  `cmd/gc/city_runtime_test.go`, `internal/beads/*ready*` tests, and the exec
  provider `ready --include-ephemeral` contract in `internal/beads/exec`.
- Operator-facing docs and historical inputs include
  `engdocs/contributors/dolt-regression-audit.md`,
  `engdocs/archive/analysis/feature-parity.md`,
  `engdocs/archive/analysis/gastown-upstream-audit.md`,
  `docs/reference/exec-beads-provider.md`, and
  `docs/reference/internal/beads-topology.md`.
- The reusable DoltLite pack and diagnostics live in `examples/beads-doltlite/`,
  `tools/doltlite-client/`, and `schemas/beads-doltlite/health/result.schema.json`.

## Proposed Implementation

Implement the audit as one convoy with three work lanes and a final report
fanin. Each lane should write evidence notes into the final audit report rather
than changing source code.

1. Build the evidence inventory.

   - Start from the requirement checklist and historical regression classes in
     `engdocs/contributors/dolt-regression-audit.md`.
   - Confirm every evidence path against the current checkout. Archived parity
     files are checklist inputs only; they do not prove current readiness.
   - Record the exact files, test names, and commands used for each claim.

2. Map known Dolt regression coverage.

   Cover at least these classes in a matrix with status `covered`, `partial`,
   `missing`, or `not applicable`:

   - `GC_DOLT_PORT` versus `BEADS_DOLT_PORT` drift.
   - stale runtime state and stale port-file rejection.
   - stale ambient `BEADS_*` and Dolt env stripping.
   - duplicate lifecycle actions and Dolt restart races.
   - unusable `.beads` bootstrap or stale `issues.jsonl` state.
   - orphaned Dolt SQL servers serving deleted or stale data.
   - missing `exec:gc-beads-bd` CRUD/ready behavior.
   - managed session `GC_BEADS=exec:gc-beads-bd` routing.
   - DoltLite native read/write fast path behavior and fallback boundaries.

3. Review provider-boundary isolation.

   - Classify each DoltLite/T3/fork-specific behavior by owner boundary:
     provider, runtime, config, pack, docs, or generic SDK.
   - Treat generic SDK leaks as findings when DoltLite or T3 assumptions appear
     outside provider, runtime, config, or pack boundaries.
   - Pay particular attention to `cmd/gc/providers.go`,
     `cmd/gc/store_target_exec_test.go`, `cmd/gc/hook_cross_store.go`,
     `cmd/gc/api_state_test.go`, `internal/beads/factory.go`,
     `internal/beads/exec/`, and the API/dashboard read paths that consume bead
     state.

4. Assess operational readiness.

   - Verify that lifecycle paths discover state from live process/runtime data
     rather than stale files alone.
   - Check cleanup and recovery behavior for managed Dolt server ownership,
     stale ports, deleted inodes, orphaned processes, backup/auto-export
     settings, health checks, and `gc doctor` behavior.
   - Include the `examples/beads-doltlite` health, doctor, build, and command
     scripts in the audit because they are the operator-facing pack boundary.

5. Write the final audit artifact.

   - Use `readiness-audit.md` under the same artifact root unless decomposition
     chooses a more specific filename.
   - Include sections for regression coverage, provider-boundary findings,
     operational readiness, verification commands, and follow-up gaps.
   - Update `manifest.json` with the audit report path, schema or `markdown`,
     SHA-256 hash, and jj document change ID.
   - Put only paths, schema IDs, hashes, and change IDs in bead metadata.

Convoy boundary: split the audit into parallel evidence lanes only after the
plan is approved. A separate follow-up convoy should handle any source fixes or
new tests discovered by the audit.

## Testing

Do not run the full Go test suite locally. Use focused checks tied to the audit
claims:

- `go test ./cmd/gc -run 'Test(BdRuntimeEnv|ResolvedRuntimeCityDoltTarget|DoltDriftCheck|PublishManagedDoltRuntimeState|StartManagedDolt|StopCityManagedBeadsProvider|DoctorSkipsDolt|DoDoctor.*Dolt|OpenStoreAtForCityExec|ControllerState.*Exec|CrossStore|PassthroughEnv.*Dolt|CityRuntime.*ManagedDolt)'`
- `go test ./internal/beads -run 'Test(BdStore.*Doltlite|BdStoreReady|DoltliteReadStore|DoltliteCount|NativeDoltStore|CachingStore.*Ready|CachingStore.*Stale)'`
- `go test ./internal/beads/contract -run 'Test(ResolveDoltConnectionTarget|Preflight|ProviderUsesBDContract|EnsureCanonicalConfig|EnsureCanonicalMetadata)'`
- `go test ./internal/beads/exec -run 'Test(Ready|ExecStoreConformance|GcBeadsBrReadyIncludeEphemeral)'`
- `go test ./examples/beads-doltlite -run 'TestDoltlite'`

If any focused check is too expensive or needs live Dolt infrastructure, record
that as `not run` with the reason and cite static evidence separately. Use
`TESTING.md` before adding or running any new test target.

## Rollout

The rollout is document-first:

- Produce the approved implementation plan under `default@`.
- Decompose the audit into evidence-collection beads after plan review.
- Write the final readiness audit and manifest update under the same artifact
  root.
- Create follow-up implementation beads only for gaps the audit proves with
  current evidence.
- Keep DoltLite source fixes, dashboard/API changes, and PR work out of this
  audit convoy unless a later approved task explicitly scopes them in.

## Open Questions

- No blocking questions for writing the plan.
- The audit report schema is not defined in this checkout; use `markdown` in
  `manifest.json` for the final report unless a downstream formula provides a
  stricter schema.
- Some Dolt readiness checks may require live Dolt or DoltLite binaries. If the
  local environment lacks them, the audit should record the missing prerequisite
  rather than infer readiness.
