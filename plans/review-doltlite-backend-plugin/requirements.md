---
schema: gc.build.requirements.v1
workflow:
  id: gc-wisp-e17
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
    - path: beads/gc-prv1
      hash: bead:gc-prv1
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

# Requirements: DoltLite Backend Plugin Architecture Review

## Problem Statement

Gas City needs a concrete review report for the DoltLite backend plugin
architecture before treating it as a viable replacement for the current
in-core DoltLite mapping in Beads-backed testing. The review must assess the
plugin repository at commit `a1bb3b202d9d50453ae4a31efd1163013428568a`
alongside the core Beads plugin-architecture workspace at commit
`cf8baef81adba06eb2e71dad472483c333c3d838`.

The output is a review artifact, not a source-change implementation. It must
identify architecture risks, missing contract coverage, and the changes needed
before this work is suitable for an upstream plugin architecture pull request.

## W6H

| Question | Answer |
| --- | --- |
| Who | Gas City and Beads maintainers deciding whether the DoltLite backend plugin is ready for upstream review. |
| What | A findings-first architecture review of the JSON stdio backend plugin, the DoltLite method mapping, process boundary, build/install story, and replacement viability. |
| When | Before implementation or PR-opening stages consume this workflow's artifacts. |
| Where | Review source context lives in `/data/projects/doltlite-gascity/rigs/beads-backend-doltlite-plugin` and `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture`; workflow documents live under `plans/review-doltlite-backend-plugin`. |
| Why | To decide what must change before the backend plugin architecture is ready for upstream review and whether core DoltLite support should remain temporarily. |
| How | Inspect the plugin and core workspaces, compare the backend protocol against DoltLite storage needs, and produce ordered findings with concrete recommendations. |
| How much | Cover the requested review focus areas and previously run verification, without implementing fixes. |

## User Stories

### REQ-001: Review the plugin protocol contract

As a Beads maintainer, I want the report to evaluate the JSON stdio backend
protocol/client/server contract so I can tell whether it is coherent enough for
upstream review.

Acceptance criteria:

- The report reviews protocol coverage across the plugin repo and core
  `backend/plugin` and `internal/backend/pluginprocess` paths.
- Findings call out contract gaps, ambiguity, or DTO boundary issues with
  file/module references where practical.
- The report distinguishes protocol design issues from DoltLite-specific
  implementation issues.

### REQ-002: Assess unmapped DoltStorage behavior

As a DoltLite backend maintainer, I want the report to evaluate the remaining
unmapped `DoltStorage` methods so I can understand replacement risk.

Acceptance criteria:

- The report covers `DB`, `UnderlyingDB`, `RunInTransaction`, and `Sync`.
- Each method is classified by severity or replacement impact.
- The report explains transaction and sync semantics risks instead of treating
  missing mappings as simple checklist items.

### REQ-003: Decide whether the plugin can replace in-core DoltLite for Gas City testing

As a Gas City maintainer, I want a recommendation on replacement viability so I
can decide whether to keep the in-core DoltLite mapping while the plugin
contract stabilizes.

Acceptance criteria:

- The report states whether the plugin can realistically replace in-core
  DoltLite for Gas City testing today.
- The recommendation accounts for live `bd` command smoke coverage already run:
  config, show, ready, stats, blocked, comments, and merge-slot checks.
- The report includes a separate recommendation on whether core DoltLite should
  stay built in temporarily.

### REQ-004: Review install and build coupling

As a release maintainer, I want the report to assess the build/install story so
I can avoid shipping a plugin that fails because `bd` and the plugin binary load
different DoltLite libraries.

Acceptance criteria:

- The report covers the requirement that `bd` and the plugin binary use the
  same `libdoltlite`.
- Findings mention any packaging, environment, or linked-build assumptions that
  would make local or CI use fragile.
- Recommendations are concrete enough to turn into implementation tasks.

### REQ-005: Evaluate switch/proxy mapping, process lifetime, and error handling risks

As a reviewer, I want the report to identify maintainability and runtime risks
from large proxy mappings and process boundaries.

Acceptance criteria:

- The report evaluates large switch/proxy mappings and whether they are
  auditable enough for upstream review.
- The report covers process lifetime, startup/shutdown, and error propagation
  risks across the plugin process boundary.
- Findings are ordered by severity and include enough context for a follow-up
  implementer to reproduce or inspect the issue.

### REQ-006: Produce a structured review report artifact

As a downstream workflow consumer, I want the requirements to force a
findings-first report shape so later stages can review or act on it without
guessing the intended output.

Acceptance criteria:

- The final report lists findings ordered by severity.
- The final report includes a short upstream-readiness recommendation.
- The final report includes a short recommendation on retaining in-core DoltLite
  while the plugin contract stabilizes.
- The report records notable verification already run and any important
  verification still missing.

## Technical Stories

### REQ-001: Protocol/client/server review

Inspect the plugin protocol and core plugin-process integration together. The
review should treat the JSON stdio boundary as the contract under test and
separate transport concerns from storage-method semantics.

### REQ-002: DoltStorage semantic coverage

Compare remaining unmapped DoltStorage methods against the behavior Gas City
and Beads need from DoltLite-backed stores. Pay particular attention to whether
transaction and sync semantics can be represented safely through the plugin
process boundary.

### REQ-004: Linked build and install review

Assess how the plugin binary and `bd` locate and link `libdoltlite`, including
whether the current approach is reproducible for local testing and eventual CI
or upstream reviewer use.

## Behavior Requirements

| ID | Requirement | Status |
| --- | --- | --- |
| REQ-001 | The review evaluates the JSON stdio backend protocol, client, and server contract. | required |
| REQ-002 | The review covers `DB`, `UnderlyingDB`, `RunInTransaction`, and `Sync` semantics. | required |
| REQ-003 | The review recommends whether the plugin can replace in-core DoltLite for Gas City testing. | required |
| REQ-004 | The review assesses install/build coupling around shared `libdoltlite` use. | required |
| REQ-005 | The review evaluates proxy mapping, process lifetime, DTO, and error-handling risks. | required |
| REQ-006 | The final artifact is findings-first and includes upstream-readiness and temporary-core-DoltLite recommendations. | required |

## Example Mapping

| Source input | Requirement coverage |
| --- | --- |
| Plugin repo `/data/projects/doltlite-gascity/rigs/beads-backend-doltlite-plugin` at `a1bb3b202d9d50453ae4a31efd1163013428568a` | REQ-001, REQ-002, REQ-004, REQ-005 |
| Core workspace `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture` on `feat/backend-plugin-architecture` at `cf8baef81adba06eb2e71dad472483c333c3d838` | REQ-001, REQ-003, REQ-005 |
| Known core tests: `go test -tags "libsqlite3 gms_pure_go" ./backend/plugin ./internal/backend/pluginprocess` | REQ-001, REQ-003 |
| Known plugin tests: `DOLTLITE_LIB="$LIB" go test -tags "libsqlite3 gms_pure_go" ./...` | REQ-002, REQ-004 |
| Known live Gas City plugin exercise: config, show, ready, stats, blocked, comments, merge-slot check | REQ-003, REQ-006 |

## Acceptance Criteria

| ID | Acceptance criteria | Coverage |
| --- | --- | --- |
| REQ-001 | Protocol/client/server coverage is reviewed with concrete findings and contract gaps separated from DoltLite-specific issues. | covered |
| REQ-002 | The remaining unmapped DoltStorage methods are reviewed for semantic and replacement risk. | covered |
| REQ-003 | The report makes an explicit replacement-viability recommendation for Gas City testing. | covered |
| REQ-004 | The report explains linked-build and install risks around shared `libdoltlite` use. | covered |
| REQ-005 | The report evaluates proxy mapping, process lifetime, DTO boundary, and error propagation risks. | covered |
| REQ-006 | The report is findings-first and includes upstream-readiness, temporary-core-DoltLite, and verification recommendations. | covered |

## Out Of Scope

- Implementing plugin, core Beads, or Gas City source changes.
- Opening or updating an upstream pull request.
- Re-running every known verification command unless a later review stage needs
  current evidence.
- Redesigning the backend plugin architecture beyond concrete recommendations
  supported by the inspected code.

## Open Questions

- Which upstream repository and reviewer expectations should govern the final
  PR-readiness bar?
- Should later implementation work prioritize filling unmapped DoltStorage
  methods first, or proving the process/lifecycle contract under longer-running
  Gas City workloads first?
