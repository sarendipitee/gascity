# DoltLite Gas City VCS Mirroring Audit Subject

Audit target: `/data/projects/doltlite-gascity/gascity`

City under test: `/data/projects/doltlite-gascity`

Clarified scope:

Audit the Gas City DoltLite integration in `gascity`, specifically whether it mirrors the VCS and workflow behavior already implemented for the Dolt-backed Gas City path. This is not primarily a `beads-doltlite` storage-backend audit.

Current Gas City stack of interest:

- `vorpomls` / `ea347098` - `source: Fix DoltLite graph locks and pool scaling`
- `lwquxpwv` / `c2f23e8a` - `source: Stop order dispatch validating beads.role`
- `qlwzstls` / `88e959ee` - `Add DoltLite client docs and skill`
- `vvunooqs` / `4b8eeb51` - `fix: declare and enforce beads-doltlite health json contract`
- `vrqmuozt` / `4c670afd` - `fix: stabilize doltlite health command JSON contract`
- `luxtzzqy` / `a10642a1` - `Guard core Dolt orders for Doltlite`
- `pytvvkxo` / `1be03d2e` - `Route DoltLite writes through fastpath`

Requested audit:

Compare the existing Dolt-backed VCS/workflow integration against the DoltLite-backed city integration and report every place where DoltLite should mirror Dolt behavior but does not.

Focus areas:

- Pack imports and agent availability:
  - The `gascity` rig has the `gc.*` workflow roles, including `gc.run-operator`, `gc.gap-analyst`, review, planning, implementation, and publication roles.
  - The `beads-doltlite` rig currently has only base provider/packer/control agents and lacks the same `gc.*` workflow role imports.
  - Audit whether the DoltLite rig should import the same packs used by the `gascity` rig so DoltLite-specific work can run its own formula roles instead of depending on the `gascity` rig.
- Dolt VCS behavior to mirror:
  - branch metadata ownership such as `metadata.branch`, `target`, `base_branch`, and `target_branch`;
  - branch creation and current-branch validation for worker/refinery flows;
  - branch-ready handoff contracts;
  - false-completion prevention using real diff checks rather than commit-count only;
  - PR/merge/refinery behavior and guards around empty or net-zero branches;
  - default branch resolution in `internal/sling`;
  - source workflow tracking and source-change metadata;
  - any Dolt maintenance/order assumptions that DoltLite must skip, replace, or emulate.
- DoltLite-specific integration surfaces:
  - `examples/beads-doltlite`;
  - `examples/bd/dolt` behavior used as the Dolt baseline;
  - `internal/storehealth`;
  - `internal/doctor` checks for DoltLite;
  - `internal/sling` DoltLite lock recovery and routing behavior;
  - `cmd/gc` rig/session/reconciler behavior under `[beads] backend = "doltlite"`;
  - city-level `pack.toml`, `city.toml`, `packs.lock`, and generated pack imports.
- Operational safety:
  - no dependency on a Dolt SQL server, Dolt runtime port, or `.gc/runtime/packs/dolt/dolt-state.json` for DoltLite cities;
  - no heavyweight DoltLite GC/flatten during startup;
  - safe behavior during live city runs before rebuilding linked `gc`, `bd`, or `doltlite-client` binaries.

Deliver a gap-analysis report that identifies:

- Dolt behavior already mirrored correctly by DoltLite integration.
- Dolt behavior that DoltLite intentionally should not mirror.
- Missing DoltLite parity in Gas City workflow/VCS behavior.
- Missing pack imports or role definitions needed by the `beads-doltlite` rig.
- Missing tests or weak tests.
- Recommended follow-up beads, grouped by implementation boundary.
