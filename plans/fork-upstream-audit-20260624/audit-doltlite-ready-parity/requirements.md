---
schema: gc.build.requirements.v1
workflow:
  id: gc-09rm
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
    - path: beads/gc-09rm
      hash: bead:gc-09rm
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

# Requirements: DoltLite Ready Parity Audit

## Problem Statement

Gas City is carrying fork-local work for a DoltLite-backed beads store while
trying to remain easy to rebase against upstream Gas City. The readiness signal
is currently spread across archived upstream parity notes, the Dolt regression
audit, provider boundary docs, tests, and command behavior. The project needs a
focused audit that answers whether the current branch is ready enough to keep
using DoltLite-backed beads for live work while build documents continue to live
in the default@ jj document workspace.

The audit must be evidence-based. Archived plans and regression notes can guide
the checklist, but every readiness claim needs current-repo confirmation. Missing
coverage, stale assumptions, or behavior that has drifted from upstream should
be reported as follow-up work, not silently treated as complete.

## W6H

| Question | Answer |
|---|---|
| Who | Gas City maintainers, workflow operators, and future implementation workers who need to know whether DoltLite-backed beads are ready for this fork. |
| What | A readiness and parity audit for DoltLite-backed beads, provider routing, lifecycle behavior, and known Dolt regression coverage. |
| Why | To prevent a fork-specific DoltLite integration from hiding regressions, upstream drift, or provider-boundary violations before source work is planned. |
| Where | The live bead database remains DoltLite-backed; audit documents are written under `plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity` in the default@ workspace. |
| When | Before implementation planning or decomposition for any DoltLite readiness fixes. |
| How | Inspect current code, tests, docs, and selected history; compare them against known Dolt regressions, upstream parity notes, and the beads provider contract. |
| How many | Cover every known Dolt regression class and every current provider boundary that can affect bead CRUD, ready queries, lifecycle, health, or cleanup. |

## User Stories

### REQ-001: Audit known Dolt regression coverage

As a Gas City maintainer, I need the audit to map each known Dolt regression to
current code and tests so readiness is based on verifiable coverage rather than
archived intent.

Acceptance criteria:

- The audit covers at least the regression classes documented for port/env drift,
  stale runtime state, duplicate lifecycle actions, unusable `.beads` bootstrap
  state, orphaned Dolt SQL servers, missing exec-provider CRUD, and managed
  session `GC_BEADS` routing.
- Each regression is marked `covered`, `partial`, `missing`, or `not applicable`.
- Each status includes concrete evidence paths such as code files, docs, tests,
  or command outputs from the current checkout.
- Any missing or uncertain evidence is reported as a follow-up gap instead of
  being inferred from archived documents.

### REQ-002: Separate live beads from jj-managed workflow documents

As a workflow operator, I need the audit workflow to keep live work in the bead
store and durable documents in default@ so downstream graph steps can continue
through manifest-managed handoff.

Acceptance criteria:

- The requirements document is a normal file under the default@ artifact root.
- `manifest.json` records the requirements path, schema, SHA-256 hash, and jj
  change ID.
- Bead metadata stores only paths, schemas, hashes, and change IDs, not document
  bodies.
- The audit distinguishes DoltLite-backed live bead storage from checked-in
  planning artifacts.

### REQ-003: Check provider-boundary isolation

As an upstream-alignment maintainer, I need the audit to identify DoltLite or
T3-specific assumptions that leaked into generic SDK paths so the fork remains
rebasing-friendly.

Acceptance criteria:

- The audit checks provider selection, bead store routing, exec-provider
  behavior, bd/Dolt lifecycle helpers, and API/dashboard read paths where they
  interact with beads.
- The audit flags assumptions that belong behind provider, runtime, config, or
  pack boundaries.
- The audit does not require a hardcoded role name, role-specific Go branch, or
  source-level judgment rule to make SDK infrastructure work.
- Fork-specific behavior is classified by ownership boundary: provider,
  runtime, config, pack, docs, or generic SDK.

### REQ-004: Assess operational readiness

As a Gas City operator, I need the audit to verify that DoltLite-backed beads can
survive common operational failure modes without stale state or silent data
loss.

Acceptance criteria:

- The audit checks port discovery, endpoint inheritance, environment
  sanitization, lifecycle restart races, health checks, cleanup, backup, and
  doctor diagnostics where current code exposes them.
- The audit identifies which focused tests or commands exercise each operational
  concern.
- The audit reports any readiness gap that would make `bd` operations, `gc hook`
  claims, graph.v2 dispatch, or API bead reads unreliable.
- The audit does not require running the full local test suite.

### REQ-005: Produce implementation-ready findings without implementing them

As a downstream planner, I need the audit output to be structured enough to drive
an implementation plan while keeping this requirements step free of source
changes.

Acceptance criteria:

- The audit output classifies findings by severity, affected boundary, evidence,
  and recommended next action.
- Follow-up candidates are grouped as source fixes, tests, documentation, or
  no-op confirmations.
- The requirements step does not create implementation beads, alter source code,
  or open a PR.
- Any recommendation that depends on historical code first points to the current
  file or commit range that should be inspected.

### REQ-006: Preserve the inherited build graph contract

As a graph.v2 workflow runner, I need the audit requirements to be valid
jj-managed build artifacts so inherited planning, decomposition, review, and
implementation steps can consume them.

Acceptance criteria:

- `requirements.md` declares `schema: gc.build.requirements.v1`.
- `manifest.json` remains valid JSON and keeps the existing workflow root,
  artifact root, and default@ workspace fields.
- The jj working copy change for the document has a non-placeholder
  description before file edits land in it.
- The workflow root's latest `gc.docs.change_id` can be updated to the jj
  change that contains the requirements artifact.

## Technical Stories

### REQ-002: Manifest-backed document handoff

The requirements artifact must be discoverable through `manifest.json` rather
than through bead comments or prompt context. Later graph steps should be able
to open the manifest, find the `requirements` document entry, verify its hash,
and continue without asking the user for paths.

### REQ-003: Provider-boundary evidence review

The audit should treat `internal/beads`, `cmd/gc/providers.go`,
`cmd/gc/beads_provider_lifecycle.go`, the exec beads provider docs, and
Dolt-related tests as primary evidence. Archived parity docs are checklist
inputs only; they do not prove current readiness.

### REQ-004: Operational failure-mode checklist

The audit should include the Dolt regression classes from
`engdocs/contributors/dolt-regression-audit.md`, with current confirmation for
port coherence, stale environment stripping, managed-session routing, lifecycle
deduplication, cleanup, backup, and doctor behavior.

## Behavior Requirements

| ID | Requirement | Status |
|---|---|---|
| REQ-001 | Known Dolt regressions are mapped to current evidence and readiness status. | covered |
| REQ-002 | Live bead storage and jj-managed documents remain separate surfaces. | covered |
| REQ-003 | Provider-boundary leaks are identified with concrete ownership boundaries. | covered |
| REQ-004 | Operational readiness checks include stale state, ports, env, lifecycle, health, and cleanup. | covered |
| REQ-005 | Findings are structured for later planning without source edits in this step. | covered |
| REQ-006 | The artifact satisfies the jj-build requirements schema and manifest handoff. | covered |

## Example Mapping

| Input | Output |
|---|---|
| `beads/gc-09rm` | `plans/fork-upstream-audit-20260624/audit-doltlite-ready-parity/requirements.md` |
| `engdocs/contributors/dolt-regression-audit.md` | Regression checklist for current evidence review |
| `engdocs/archive/analysis/feature-parity.md` | Upstream parity checklist input, not proof of current state |
| `docs/reference/internal/beads-topology.md` | Operational topology evidence for city and rig bead storage |
| `docs/reference/exec-beads-provider.md` | Provider-boundary evidence for pluggable bead storage |

## Acceptance Criteria

| ID | Status |
|---|---|
| REQ-001 | The audit has a regression coverage matrix with evidence and gaps. |
| REQ-002 | The requirements document and manifest are updated under the default@ artifact root. |
| REQ-003 | Any DoltLite/T3-specific generic-SDK leakage is named with an owner boundary. |
| REQ-004 | Operational checks cover port, env, lifecycle, health, cleanup, and API/read behavior where applicable. |
| REQ-005 | No source implementation, bead creation, or PR work is performed by the requirements step. |
| REQ-006 | Downstream jj-build steps can discover the requirements file through `manifest.json`. |

## Out Of Scope

- Implementing DoltLite, bd, exec-provider, API, dashboard, or T3 bridge fixes.
- Creating implementation beads or launching implementation/review formulas.
- Running the full local Go test suite.
- Rewriting archived audit documents.
- Moving workflow documents out of the default@ artifact root.

## Open Questions

No blocking open questions for the requirements step. The audit itself may
surface follow-up questions when current code and historical parity notes
disagree.
