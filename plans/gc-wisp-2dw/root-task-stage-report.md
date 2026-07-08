# Root Task Stage Report

Generated: 2026-07-04T09:09:15.030Z
City root: /data/projects/doltlite-gascity/gascity

## Filters

- Status: open, in_progress, blocked, deferred
- Types: task, bug, feature, chore
- Root only: parent/root fields must be empty
- Workflow metadata beads: excluded when gc.kind is set
- Wisps/orders: excluded by id/metadata/title

## Summary

| Rig | Total | Stages |
|---|---:|---|
| beads-doltlite | 15 | decomposition: 1, finalization/reporting: 8, planning: 2, publication: 1, review: 3 |
| gascity | 75 | bug triage: 11, finalization/reporting: 1, implementation/cleanup: 26, review: 13, unsorted: 18, verification: 6 |
| gascity-packs | 7 | implementation/cleanup: 5, review: 1, unsorted: 1 |
| lightjj | 21 | decomposition: 1, finalization/reporting: 11, planning: 2, publication: 2, requirements: 2, review: 2, unsorted: 1 |
| beads-backend-doltlite-plugin | 1 | unsorted: 1 |
| **All rigs** | **119** | bug triage: 11, decomposition: 2, finalization/reporting: 20, implementation/cleanup: 31, planning: 4, publication: 3, requirements: 2, review: 19, unsorted: 21, verification: 6 |

## beads-doltlite

| ID | Status | Type | Stage | Assignee | Title |
|---|---|---|---|---|---|
| bd-07h | open | task | publication | gc__publisher-dg-wisp-xwutzgf | Optionally publish jj build result |
| bd-ara | open | task | finalization/reporting | gc__run-operator-dg-wisp-6tz55h | Finalize jj-managed build result |
| bd-8gf | open | task | finalization/reporting | gc__run-operator-dg-wisp-6tz55h | Describe final report document change |
| bd-rst | open | task | review | gc__implementation-reviewer-dg-wisp-6qibpp1 | Review jj implementation |
| bd-dkr | open | task | finalization/reporting | gc__implementation-reviewer-dg-wisp-6qibpp1 | Describe review document change |
| bd-d82 | open | task | finalization/reporting | gc__run-operator-dg-wisp-6tz55h | Write jj-managed implementation summary |
| bd-hix | open | task | finalization/reporting | gc__run-operator-dg-wisp-6tz55h | Describe implementation summary document change |
| bd-141 | open | task | decomposition | gc__task-decomposer-dg-wisp-uu9dxzw | Decompose approved jj-managed plan |
| bd-9eq | open | task | finalization/reporting | gc__task-decomposer-dg-wisp-uu9dxzw | Describe decomposition document change |
| bd-wyt | open | task | planning | gc__review-synthesizer-dg-wisp-pq3x2nv | Review jj-managed implementation plan |
| bd-54u | open | task | finalization/reporting | gc__review-synthesizer-dg-wisp-pq3x2nv | Describe plan review document change |
| bd-2y2 | open | task | planning | gc__design-author-dg-wisp-lach0ab | Produce jj-managed implementation plan |
| bd-ieo | open | task | finalization/reporting | gc__design-author-dg-wisp-lach0ab | Describe plan document change |
| bd-5tj | open | task | review | duncan4123@users.noreply.github.com | Critically audit controller/formula DoltLite query-write coverage |
| bd-8lz | open | task | review | duncan4123@users.noreply.github.com | Audit beads-doltlite default@ jj line |

## gascity

| ID | Status | Type | Stage | Assignee | Title |
|---|---|---|---|---|---|
| gc-vquz.2 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Repair order dispatch tracking creation issue_prefix failures |
| gc-vquz.1 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Fix order tracking bead scope and missing-bead lifecycle failures |
| gc-vquz | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Implement order-system recovery from Mayor triage dg-zcr |
| gc-1d30 | open | task | unsorted | duncan4123@users.noreply.github.com | Implement raw SQL support for the DoltLite Beads backend plugin |
| gc-4y91 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Fix bd list --include-ephemeral no-history workflow visibility |
| gc-vquz.7 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Fix legacy Dolt order leakage into DoltLite city imports |
| gc-vquz.4 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Bound or clear lingering orphan-sweep order process |
| gc-vquz.3 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Investigate malformed gascity-dashboard DoltLite bead database |
| gc-pscd.8.2 | open | task | unsorted | duncan4123@users.noreply.github.com | pr-pipeline: port workflows from git/polecat assumptions to current jj graph.v2 workflow |
| gc-pscd.12 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Fix packsmith workspace materialization and lint command resolution |
| gc-pscd.10 | open | task | unsorted | duncan4123@users.noreply.github.com | Make jj-pack formulas validate and run |
| gc-pscd.9 | open | task | bug triage | duncan4123@users.noreply.github.com | Fix gascity-jj-base lint failures |
| gc-pscd | open | task | review | duncan4123@users.noreply.github.com | Deep audit current installed pack setup |
| gc-2w4f | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Verify and fix Codex session hooks wall-of-text issue in build f30c7ecb8da6dd44e11ff8508c40fab214585609 |
| gc-jyqp | open | task | review | duncan4123@users.noreply.github.com | Clean gascity-review-split JJ history |
| gc-f0vg | open | task | review | duncan4123@users.noreply.github.com | Critically audit controller/formula DoltLite query-write coverage |
| gc-m6j2 | open | task | finalization/reporting | duncan4123@users.noreply.github.com | Write DoltLite readiness audit report |
| gc-vkeh | open | task | review | duncan4123@users.noreply.github.com | Audit DoltLite provider boundaries and operations |
| gc-1m8v | open | task | unsorted | duncan4123@users.noreply.github.com | Map Dolt regression coverage |
| gc-1ljf | open | task | review | duncan4123@users.noreply.github.com | Audit DoltLite readiness evidence inventory |
| gc-a3ij | open | bug | unsorted | duncan4123@users.noreply.github.com | Track tmux 3.4 native crash evidence |
| gc-r1br | open | task | review | duncan4123@users.noreply.github.com | Audit DoltLite init fastpath setup |
| gc-prv1 | open | task | review | duncan4123@users.noreply.github.com | Review DoltLite backend plugin architecture |
| gc-v6bo | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | gascity-jj-base: packsmith smoke workspace |
| gc-krrt | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | gascity-jj-base: packsmith smoke workspace |
| gc-gemc | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | gascity-jj-base: packsmith smoke workspace |
| gc-r304 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | gascity-jj-base: packsmith smoke workspace |
| gc-vquz.6 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Route control-dispatcher ready queries through DoltLite-safe fastpath |
| gc-vquz.5 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Stop Beads telemetry flusher from amplifying Gas City internal query load |
| gc-4i36 | open | task | unsorted | duncan4123@users.noreply.github.com | Restore bd pack testability by providing required go.sum entries |
| gc-5ovt | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Make packsmith lint use routed pack_root for bd pack workspaces |
| gc-pscd.4.2 | open | task | unsorted | duncan4123@users.noreply.github.com | Make gc lint resolve installed path imports for core |
| gc-pscd.4.1 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Fix packsmith routing for bootstrap core pack workspaces |
| gc-pscd.3.2 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Align future DoltLite init pack pins with current installed pack.toml |
| gc-pscd.3.1 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Fix packsmith routing for beads-doltlite-init bootstrap pack audits |
| gc-pscd.8.1 | open | task | unsorted | duncan4123@users.noreply.github.com | pr-pipeline: make remote import catalog/lint checks first-class |
| gc-pscd.11 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Align packsmith route target defaults with installed rig routes |
| gc-pscd.7.2 | open | task | unsorted | duncan4123@users.noreply.github.com | jj-hunk: retire or label stale ./packs copy |
| gc-pscd.7.1 | open | task | unsorted | duncan4123@users.noreply.github.com | jj-hunk: standardize active import source |
| gc-t692 | in_progress | task | implementation/cleanup | Claude Opus 4.6 | beads-doltlite: align DoltLite fork with PR 3800 front-door seams |
| gc-kd0i | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Verify PR 3617 review fixes and related notes |
| gc-0hqq | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Resolve gc.pack_root metadata consumption |
| gc-xtog | open | task | unsorted | duncan4123@users.noreply.github.com | Replace packer-specific core environment variables |
| gc-7enm | open | task | verification | duncan4123@users.noreply.github.com | Verify DoltLite lock and routed-pool regressions |
| gc-duxe | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Fix ready routed target pool convergence |
| gc-qep8 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Fix DoltLite graph creation lock recovery |
| gc-b7ta | open | task | review | duncan4123@users.noreply.github.com | Audit NativeDoltStore and beadslib parity |
| gc-8etk | open | task | review | duncan4123@users.noreply.github.com | Audit jj-pack formula integration on DoltLite |
| gc-1aqb | open | task | review | duncan4123@users.noreply.github.com | Audit bd bridge and controller query parity |
| gc-bn35 | open | task | review | duncan4123@users.noreply.github.com | Audit DoltLite direct read store parity |
| gc-ycoj | open | task | verification | duncan4123@users.noreply.github.com | Verify first-start readiness and focused gates |
| gc-na8m | open | task | unsorted | duncan4123@users.noreply.github.com | Preserve DoltLite bridge environment |
| gc-u5q6 | open | task | unsorted | duncan4123@users.noreply.github.com | Consolidate DoltLite backend and provider detection |
| gc-gwqj | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Fix DoltLite init persistence and provider readiness |
| gc-f512 | open | task | unsorted | duncan4123@users.noreply.github.com | Add DoltLite init selection regression tests |
| gc-53gd | in_progress | task | implementation/cleanup | Claude Opus 4.6 | Remove unused gc.pack_root metadata key |
| gc-d7zx | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Fix missing jjw-workspace-report order override warning |
| gc-waiz | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Expose formula metadata in gc formula show JSON |
| gc-o05t | open | bug | bug triage | duncan4123@users.noreply.github.com | GitHub PR review workflow loses review artifacts before posting |
| gc-04yo | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Address PR #3617 review changes |
| gc-6jad | open | task | review | duncan4123@users.noreply.github.com | Audit gascity default@ jj line |
| gc-vzbu | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Set up clean DoltLite Gas City build workspace |
| gc-30e8 | open | task | review | duncan4123@users.noreply.github.com | Review DoltLite demand snapshot supervisor design |
| gc-9b3z | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Fix DoltLite integration gaps from completed gap analyses |
| gc-2zf | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Smoke test gascity-jj-base source workspace lane |
| gc-cpr2 | open | task | unsorted | duncan4123@users.noreply.github.com | Resolve conflicted upstream/object-front-doors jj bookmark |
| gc-s552 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Align bd pack workspace sparse shared-infra patterns with actual files |
| gc-h1z4 | open | task | unsorted | duncan4123@users.noreply.github.com | Reconcile active bd local path import with pack lock and registry expectations |
| gc-pscd.4.3 | open | task | unsorted | duncan4123@users.noreply.github.com | Decide catalog visibility for bundled core formulas |
| gc-pscd.3.3 | open | task | unsorted | duncan4123@users.noreply.github.com | Document and harden beads-doltlite-init validation commands |
| gc-pscd.7.3 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | jj-hunk: align registry release metadata |
| gc-8j23 | open | task | verification | duncan4123@users.noreply.github.com | Verify jj-build manifest-backed handoff |
| gc-l8gl | open | task | verification | duncan4123@users.noreply.github.com | Smoke: linked DoltLite jj-build sling |
| gc-51fi | open | task | verification | duncan4123@users.noreply.github.com | Smoke: linked DoltLite jj-review sling |
| gc-fsmoke-181041 | open | task | verification | duncan4123@users.noreply.github.com | Formula plugin trace smoke |

## gascity-packs

| ID | Status | Type | Stage | Assignee | Title |
|---|---|---|---|---|---|
| gp-ojkf | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Route tmux hook popup fix for DoltLite rigs |
| gp-2ocp | open | task | unsorted | duncan4123@users.noreply.github.com | Route pack-aware JJ formula wiring |
| gp-ej0f | open | task | review | duncan4123@users.noreply.github.com | Audit gascity-packs default@ jj line |
| gp-2z8l | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Route gascity-jj-base README setup alignment |
| gp-rimc | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Align fork packs with upstream pack patterns |
| gp-gnen | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Coordinate pack workspace integration sweep |
| gp-qmh | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Align active gascity packs with upstream pack patterns |

## lightjj

| ID | Status | Type | Stage | Assignee | Title |
|---|---|---|---|---|---|
| lj-qzj | open | task | publication | gc__publisher-dg-wisp-323a4nf | Optionally publish jj build result |
| lj-0pz | open | task | finalization/reporting | gc__run-operator-dg-wisp-cgeahz | Finalize jj-managed build result |
| lj-6x8 | open | task | finalization/reporting | gc__run-operator-dg-wisp-cgeahz | Describe final report document change |
| lj-x13 | open | task | review | gc__implementation-reviewer-dg-wisp-28cgjdb | Review jj implementation |
| lj-zeh | open | task | finalization/reporting | gc__implementation-reviewer-dg-wisp-28cgjdb | Describe review document change |
| lj-ci4 | open | task | finalization/reporting | gc__run-operator-dg-wisp-cgeahz | Write jj-managed implementation summary |
| lj-1gi | open | task | finalization/reporting | gc__run-operator-dg-wisp-cgeahz | Describe implementation summary document change |
| lj-3f5 | open | task | decomposition | gc__task-decomposer-dg-wisp-4ts5yp6 | Decompose approved jj-managed plan |
| lj-lk0 | open | task | finalization/reporting | gc__task-decomposer-dg-wisp-4ts5yp6 | Describe decomposition document change |
| lj-mdy | open | task | planning | gc__review-synthesizer-dg-wisp-k8xv265 | Review jj-managed implementation plan |
| lj-54i | open | task | finalization/reporting | gc__review-synthesizer-dg-wisp-k8xv265 | Describe plan review document change |
| lj-ccz | open | task | planning | gc__design-author-dg-wisp-5pytf24 | Produce jj-managed implementation plan |
| lj-1jf | open | task | finalization/reporting | gc__design-author-dg-wisp-5pytf24 | Describe plan document change |
| lj-725 | open | task | requirements | gc__requirements-planner-dg-wisp-a487id1 | Produce jj-managed requirements |
| lj-m79 | open | task | requirements | gc__requirements-planner-dg-wisp-a487id1 | Describe requirements document change |
| lj-zg3 | open | feature | unsorted | duncan4123@users.noreply.github.com | Implement Taplo-backed TOML schema editing in LightJJ |
| lj-0mb | open | task | publication | gc__publisher-dg-wisp-323a4nf | Optionally publish jj build result |
| lj-vex | open | task | finalization/reporting | gc__run-operator-dg-wisp-z7rkaf0 | Finalize jj-managed build result |
| lj-l37 | open | task | finalization/reporting | gc__run-operator-dg-wisp-bbtcvif | Describe final report document change |
| lj-9i5 | open | task | review | gc__implementation-reviewer-dg-wisp-9w9d6k0 | Review jj implementation |
| lj-dyp | open | task | finalization/reporting | gc__implementation-reviewer-dg-wisp-4gprlym | Describe review document change |

## beads-backend-doltlite-plugin

| ID | Status | Type | Stage | Assignee | Title |
|---|---|---|---|---|---|
| bdp-p17 | open | task | unsorted | duncan4123@users.noreply.github.com | Prototype DoltLite backend plugin process contract |
