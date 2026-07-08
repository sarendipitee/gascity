---
schema: gc.build.plan.v1
workflow:
  id: gc-wisp-e17
  formula: jj-build
methodology:
  pack: gascity-jj-base
  name: jj-build
producer:
  formula: jj-build
  stage: plan
  attempt: 1
status: approved
trace:
  upstream:
    - path: plans/review-doltlite-backend-plugin/requirements.md
      hash: sha256:cee760d53d50acffb0157b1c691fa3395aa5b0ddb6e327cc76d2ac0b4553690b
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

# Implementation Plan: DoltLite Backend Plugin Architecture Review

## Summary

Produce a findings-first architecture review report for the DoltLite backend
plugin work. The implementation work is document-only: inspect the plugin and
core Beads plugin-architecture workspaces, compare the JSON stdio backend
contract against DoltLite-backed storage requirements, and write an ordered
review artifact with replacement-viability and upstream-readiness
recommendations.

The review must not implement source changes, open a pull request, or redesign
the backend plugin system. Its value is a concrete decision record that later
implementation work can convert into tasks.

## Current System

- Workflow documents are tracked under
  `plans/review-doltlite-backend-plugin` in the Gas City default jj workspace.
- The approved requirements are in
  `plans/review-doltlite-backend-plugin/requirements.md` and require coverage
  of `REQ-001` through `REQ-006`.
- The review source context is split across two workspaces:
  `/data/projects/doltlite-gascity/rigs/beads-backend-doltlite-plugin` for the
  DoltLite plugin implementation, and
  `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture` for
  the core Beads backend plugin architecture.
- The requirements name the plugin source commit
  `a1bb3b202d9d50453ae4a31efd1163013428568a` and the core source commit
  `cf8baef81adba06eb2e71dad472483c333c3d838` as the intended review inputs.
  The reviewer should verify and record the actual inspected revisions before
  drawing conclusions.
- Known focus paths include `backend/plugin/protocol.go`,
  `internal/backend/pluginprocess`, plugin command entry points under
  `cmd/bd-backend-doltlite` and `cmd/gc-doltlite*`, and DoltLite storage
  implementation files under `internal/storage`.
- Known prior verification covers core plugin tests, plugin tests, and live Gas
  City `bd` command smoke coverage for config, show, ready, stats, blocked,
  comments, and merge-slot checks. The report should treat that as evidence to
  cite, not as a substitute for reviewing the architecture risks.

## Proposed Implementation

Create one review-report task convoy with the following work boundaries:

1. Source-state confirmation:
   - Verify both source workspaces exist and record their current revisions,
     branch/bookmark state, and any dirty working-copy state.
   - If either workspace is not at the requirements-named commit, continue only
     by recording the mismatch clearly in the report's verification section.
   - Do not copy source files into the document artifact; cite paths and
     revisions instead.

2. Protocol and process-boundary review:
   - Inspect `backend/plugin/protocol.go` in both workspaces and the core
     `internal/backend/pluginprocess` integration.
   - Review request/response DTO shape, error propagation, process startup,
     shutdown, and lifecycle behavior.
   - Separate generic backend-plugin contract findings from DoltLite-specific
     plugin findings.

3. DoltStorage semantic coverage review:
   - Inspect the plugin storage implementation under `internal/storage`.
   - Classify the replacement impact of unmapped or partially mapped
     `DoltStorage` behavior, specifically `DB`, `UnderlyingDB`,
     `RunInTransaction`, and `Sync`.
   - Treat transaction and sync behavior as semantic risks, not as simple
     missing-method checklist items.

4. Build/install and linked-library review:
   - Review plugin command entry points and install/build documentation for how
     `bd`, the plugin binary, and `libdoltlite` are located and linked.
   - Identify assumptions that make local testing or CI fragile, especially
     cases where `bd` and the plugin process could load different DoltLite
     libraries.

5. Replacement-viability and upstream-readiness recommendation:
   - State whether the plugin can replace in-core DoltLite for Gas City testing
     today.
   - Include a separate recommendation on whether core DoltLite support should
     remain built in temporarily while the plugin contract stabilizes.
   - Summarize what must change before an upstream plugin architecture pull
     request is ready.

6. Final artifact production:
   - Write the report as a normal markdown file under
     `plans/review-doltlite-backend-plugin`, preferably `review-report.md`.
   - Use a findings-first structure: ordered findings first, then evidence,
     recommendations, verification run, and missing verification.
   - Update `manifest.json` with the report path, schema, SHA-256 hash, and jj
     document change ID when the report document is produced.

## Testing

- Validate that the plan artifact is listed in
  `plans/review-doltlite-backend-plugin/manifest.json` with schema
  `gc.build.plan.v1`, a SHA-256 content hash, and the current document jj
  change ID.
- For the later report task, verify the final report covers every requirement
  in the frontmatter trace: `REQ-001` through `REQ-006`.
- For the later report task, run or cite the focused verification commands only
  when they are needed to support a finding:
  `go test -tags "libsqlite3 gms_pure_go" ./backend/plugin ./internal/backend/pluginprocess`
  in the core workspace, and
  `DOLTLITE_LIB="$LIB" go test -tags "libsqlite3 gms_pure_go" ./...` in the
  plugin workspace.
- Record any command that is skipped, stale, or blocked as missing verification
  rather than silently treating it as passed.

## Rollout

This is a document workflow. After this plan is recorded, downstream work should
create or update only review artifacts in
`plans/review-doltlite-backend-plugin` until a separate implementation workflow
is approved. Source changes to Beads, Gas City, or the DoltLite plugin are out
of scope for this plan.

## Open Questions

- Which upstream repository and reviewer expectations should define the final
  PR-readiness bar?
- Should follow-up implementation prioritize filling the unmapped DoltStorage
  methods first, or proving the process/lifecycle contract under longer-running
  Gas City workloads first?
