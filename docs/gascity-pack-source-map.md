# Gas City Pack Source Map

This repo owns `gc`, builtin packs, and the defaults that future `gc init`
commands write. The important split is city runtime config, active pack imports,
resolved pack locks, pack registry catalogs, future `gc init` defaults, and
build/install source selection.

This copy lives in the `gascity` default workspace. It is the best starting
point when changing what a fresh DoltLite city will import.

## City Runtime

File: `/data/projects/doltlite-gascity/city.toml`

This controls the city itself:

- beads backend, currently `doltlite`
- rig imports
- role imports
- local pack roots such as `gascity-packs/gstack`,
  `gascity-packs/gascity-jj-base`, and related packs

This is not the pack registry. It tells this city what rigs, roles, and backend
to use.

## Active City Pack Imports

File: `/data/projects/doltlite-gascity/pack.toml`

This is the active pack import file for this city. This is what matters for the
current local city.

The important imports are:

- `bd`: `/data/projects/doltlite-gascity/gascity/examples/bd`
- `core`: `/data/projects/doltlite-gascity/gascity/internal/bootstrap/packs/core`
- `beads-doltlite`: `https://github.com/duncan4123/gascity-packs/tree/main/beads-doltlite`
- `beads-doltlite-init`: `https://github.com/duncan4123/gascity-packs/tree/main/beads-doltlite-init`

So for this city, the `beads-doltlite` pack has already been pointed at the
`gascity-packs` fork main branch.

## Resolved Pack Lock

File: `/data/projects/doltlite-gascity/packs.lock`

This records what remote imports resolved to. It has pinned commits for imports
such as:

- `beads-doltlite`
- `beads-doltlite-init`
- `gascity/roles`
- `pr-pipeline`

This is resolved install state. Edit the import source first, then let the lock
update from that source.

## Pack Registry Catalog

Primary file:

- `/data/projects/doltlite-gascity/gascity-packs/registry.toml`

Workspace copies that may also exist:

- `/data/projects/doltlite-gascity/gascity-packs-land-beads-doltlite/registry.toml`
- `/data/projects/doltlite-gascity/gascity-packs/.jjw/workspaces/beads-doltlite-publish/registry.toml`

The `gascity-packs/registry.toml` file is the registry catalog in the
`gascity-packs` fork. The landing and publish paths are JJ workspaces and are
not active city config unless the city points at them.

## Future `gc init` Defaults

These are inside the `gascity` source tree:

- `/data/projects/doltlite-gascity/gascity/cmd/gc/cmd_init.go`
- `/data/projects/doltlite-gascity/gascity/cmd/gc/embed_builtin_packs.go`
- `/data/projects/doltlite-gascity/gascity/cmd/gc/beads_backend.go`
- `/data/projects/doltlite-gascity/gascity/internal/builtinpacks/registry.go`
- `/data/projects/doltlite-gascity/gascity/internal/config/public_packs.go`

`internal/config/public_packs.go` is the main file for external public pack
defaults. If this file points at an old publish branch, future `gc init` output
can differ from the current local `pack.toml`.

## Build And Install Source Selection

The `beads-doltlite` pack build script controls where `gc`, `bd`, and
`doltlite-client` get built from.

Relevant file:

- `/data/projects/doltlite-gascity/gascity-packs/beads-doltlite/commands/build/run.sh`

That script has its own source resolution:

- prefers local `$CITY_ROOT/gascity`
- prefers local `$CITY_ROOT/beads-doltlite`
- otherwise fetches remotes
- default gascity remote: `https://github.com/duncan4123/gascity.git`, ref
  `pr-doltlite-foundation`
- default beads remote: `https://github.com/duncan4123/beads-doltlite.git`, ref
  `beads-doltlite`
- DoltLite lib can come from downloaded release state or local build dirs

This is separate from `pack.toml`.

## Current Mismatches To Clean Up

- The active city `pack.toml` points `beads-doltlite` to `gascity-packs` fork
  main.
- Future `gc init` defaults in `gascity/internal/config/public_packs.go` may
  still point at an older publish branch unless updated.
- The build script can pick local `$CITY_ROOT/beads-doltlite`. In this city
  that checkout is not the DoltLite-capable one.
- The DoltLite-capable city-local checkout appears to be
  `/data/projects/doltlite-gascity/beads-doltlite-ci`.

## Cleanup Target

Keep these in agreement:

- root `pack.toml`
- `gascity/internal/config/public_packs.go`
- `gascity-packs/registry.toml`
- the `beads-doltlite` build script source defaults and overrides

For normal local installs, build `bd` from the DoltLite-capable city-local
source, not from the stale `$CITY_ROOT/beads-doltlite` checkout.
