# Root Task Stage Report

Generated: 2026-07-04T13:50:01.810Z
City root: /data/projects/doltlite-gascity/gascity

## Filters

- Status: open, in_progress, blocked, deferred
- Types: task, bug, feature, chore
- Root only: parent/root fields must be empty
- Workflow metadata beads: excluded when gc.kind is set
- Wisps/orders: excluded by id/metadata/title

## Query Notes

- Skipped `gascity-packs`: bead store failed to open because DoltLite schema validation could not find `schema_migrations`.

## Summary

| Rig | Total | Stages |
|---|---:|---|
| beads-backend-doltlite-plugin | 1 | unsorted: 1 |
| beads-doltlite | 2 | review: 1, verification: 1 |
| doltlite-gascity | 8 | bug triage: 7, unsorted: 1 |
| gascity | 61 | bug triage: 11, finalization/reporting: 1, implementation/cleanup: 25, review: 12, unsorted: 1, verification: 11 |
| lightjj | 1 | implementation/cleanup: 1 |
| **All rigs** | **73** | bug triage: 18, finalization/reporting: 1, implementation/cleanup: 26, review: 13, unsorted: 3, verification: 12 |

## beads-backend-doltlite-plugin

| ID | Status | Type | Stage | Assignee | Title |
|---|---|---|---|---|---|
| bdp-p17 | open | task | unsorted | duncan4123@users.noreply.github.com | Prototype DoltLite backend plugin process contract |

## beads-doltlite

| ID | Status | Type | Stage | Assignee | Title |
|---|---|---|---|---|---|
| bd-5tj | open | task | verification | duncan4123@users.noreply.github.com | Critically audit controller/formula DoltLite query-write coverage |
| bd-8lz | open | task | review | duncan4123@users.noreply.github.com | Audit beads-doltlite default@ jj line |

## doltlite-gascity

| ID | Status | Type | Stage | Assignee | Title |
|---|---|---|---|---|---|
| dg-zcr.2 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Repair order dispatch tracking creation issue_prefix failures |
| dg-zcr.1 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Fix order tracking bead scope and missing-bead lifecycle failures |
| dg-zcr | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Stabilize Gas City order dispatcher on DoltLite city |
| dg-zcr.4 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Bound or clear lingering orphan-sweep order process |
| dg-zcr.3 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Investigate malformed gascity-dashboard DoltLite bead database |
| dg-4fx | open | bug | bug triage | duncan4123@users.noreply.github.com | Fix beads-doltlite transitive import of legacy Dolt bd pack |
| dg-zcr.5 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Stop Beads telemetry flusher from amplifying Gas City internal query load |
| dg-9ig | open | task | unsorted | duncan4123@users.noreply.github.com | Investigate routing control-dispatcher ready queries through DoltLite fastpath |

## gascity

| ID | Status | Type | Stage | Assignee | Title |
|---|---|---|---|---|---|
| gc-vquz.2 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Repair order dispatch tracking creation issue_prefix failures |
| gc-vquz.1 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Fix order tracking bead scope and missing-bead lifecycle failures |
| gc-vquz | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Implement order-system recovery from Mayor triage dg-zcr |
| gc-1d30 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Implement raw SQL support for the DoltLite Beads backend plugin |
| gc-4y91 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Fix bd list --include-ephemeral no-history workflow visibility |
| gc-vquz.7 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Fix legacy Dolt order leakage into DoltLite city imports |
| gc-vquz.4 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Bound or clear lingering orphan-sweep order process |
| gc-vquz.3 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Investigate malformed gascity-dashboard DoltLite bead database |
| gc-pscd.8.2 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | pr-pipeline: port workflows from git/polecat assumptions to current jj graph.v2 workflow |
| gc-pscd.12 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Fix packsmith workspace materialization and lint command resolution |
| gc-pscd.10 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Make jj-pack formulas validate and run |
| gc-pscd.9 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Fix gascity-jj-base lint failures |
| gc-pscd | open | task | review | duncan4123@users.noreply.github.com | Deep audit current installed pack setup |
| gc-2w4f | open | task | verification | duncan4123@users.noreply.github.com | Verify and fix Codex session hooks wall-of-text issue in build f30c7ecb8da6dd44e11ff8508c40fab214585609 |
| gc-jyqp | open | task | review | duncan4123@users.noreply.github.com | Clean gascity-review-split JJ history |
| gc-f0vg | open | task | verification | duncan4123@users.noreply.github.com | Critically audit controller/formula DoltLite query-write coverage |
| gc-a3ij | open | bug | bug triage | duncan4123@users.noreply.github.com | Track tmux 3.4 native crash evidence |
| gc-r1br | open | task | review | duncan4123@users.noreply.github.com | Audit DoltLite init fastpath setup |
| gc-9pez | open | bug | bug triage | duncan4123@users.noreply.github.com | Fix gc sling live-workflow guard DoltLite schema_migrations failure |
| gc-bh8m | open | task | verification | duncan4123@users.noreply.github.com | Smoke build-basic review for gascity DoltLite changes |
| gc-xo69 | open | task | review | duncan4123@users.noreply.github.com | Review DoltLite backend plugin integration across Beads and Gas City |
| gc-prv1 | open | task | review | duncan4123@users.noreply.github.com | Review DoltLite backend plugin architecture |
| gc-v6bo | open | task | verification | duncan4123@users.noreply.github.com | gascity-jj-base: packsmith smoke workspace |
| gc-krrt | open | task | verification | duncan4123@users.noreply.github.com | gascity-jj-base: packsmith smoke workspace |
| gc-gemc | open | task | verification | duncan4123@users.noreply.github.com | gascity-jj-base: packsmith smoke workspace |
| gc-r304 | open | task | verification | duncan4123@users.noreply.github.com | gascity-jj-base: packsmith smoke workspace |
| gc-vquz.6 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Route control-dispatcher ready queries through DoltLite-safe fastpath |
| gc-vquz.5 | open | bug | bug triage | duncan4123@users.noreply.github.com | [BUG] Stop Beads telemetry flusher from amplifying Gas City internal query load |
| gc-4i36 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Restore bd pack testability by providing required go.sum entries |
| gc-5ovt | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Make packsmith lint use routed pack_root for bd pack workspaces |
| gc-pscd.4.2 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Make gc lint resolve installed path imports for core |
| gc-pscd.4.1 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Fix packsmith routing for bootstrap core pack workspaces |
| gc-pscd.3.2 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Align future DoltLite init pack pins with current installed pack.toml |
| gc-pscd.3.1 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Fix packsmith routing for beads-doltlite-init bootstrap pack audits |
| gc-pscd.8.1 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | pr-pipeline: make remote import catalog/lint checks first-class |
| gc-pscd.11 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Align packsmith route target defaults with installed rig routes |
| gc-pscd.7.2 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | jj-hunk: retire or label stale ./packs copy |
| gc-pscd.7.1 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | jj-hunk: standardize active import source |
| gc-t692 | in_progress | task | implementation/cleanup | duncan4123@users.noreply.github.com | beads-doltlite: align DoltLite fork with PR 3800 front-door seams |
| gc-1aqb | open | task | review | duncan4123@users.noreply.github.com | Audit bd bridge and controller query parity |
| gc-8etk | open | task | review | duncan4123@users.noreply.github.com | Audit jj-pack formula integration on DoltLite |
| gc-b7ta | open | task | review | duncan4123@users.noreply.github.com | Audit NativeDoltStore and beadslib parity |
| gc-bn35 | open | task | review | duncan4123@users.noreply.github.com | Audit DoltLite direct read store parity |
| gc-53gd | in_progress | task | implementation/cleanup | duncan4123@users.noreply.github.com | Remove unused gc.pack_root metadata key |
| gc-d7zx | open | task | finalization/reporting | duncan4123@users.noreply.github.com | Fix missing jjw-workspace-report order override warning |
| gc-waiz | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Expose formula metadata in gc formula show JSON |
| gc-04yo | open | task | review | duncan4123@users.noreply.github.com | Address PR #3617 review changes |
| gc-6jad | open | task | review | duncan4123@users.noreply.github.com | Audit gascity default@ jj line |
| gc-vzbu | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Set up clean DoltLite Gas City build workspace |
| gc-30e8 | open | task | review | duncan4123@users.noreply.github.com | Review DoltLite demand snapshot supervisor design |
| gc-9b3z | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Fix DoltLite integration gaps from completed gap analyses |
| gc-2zf | open | task | verification | duncan4123@users.noreply.github.com | Smoke test gascity-jj-base source workspace lane |
| gc-cpr2 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Resolve conflicted upstream/object-front-doors jj bookmark |
| gc-s552 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Align bd pack workspace sparse shared-infra patterns with actual files |
| gc-h1z4 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Reconcile active bd local path import with pack lock and registry expectations |
| gc-pscd.4.3 | open | task | unsorted | duncan4123@users.noreply.github.com | Decide catalog visibility for bundled core formulas |
| gc-pscd.3.3 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | Document and harden beads-doltlite-init validation commands |
| gc-pscd.7.3 | open | task | implementation/cleanup | duncan4123@users.noreply.github.com | jj-hunk: align registry release metadata |
| gc-51fi | open | task | verification | duncan4123@users.noreply.github.com | Smoke: linked DoltLite jj-review sling |
| gc-l8gl | open | task | verification | duncan4123@users.noreply.github.com | Smoke: linked DoltLite jj-build sling |
| gc-fsmoke-181041 | open | task | verification | duncan4123@users.noreply.github.com | Formula plugin trace smoke |

## lightjj

| ID | Status | Type | Stage | Assignee | Title |
|---|---|---|---|---|---|
| lj-zg3 | open | feature | implementation/cleanup | duncan4123@users.noreply.github.com | Implement Taplo-backed TOML schema editing in LightJJ |
