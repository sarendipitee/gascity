# Beads Backend Plugin Handoff

Date: 2026-07-04

This document captures the important pieces of today's Beads backend plugin
work, where each piece lives, how they fit together, and what still needs to be
organized before releases.

## Goal

Move the DoltLite integration toward a plugin architecture that upstream Beads
maintainers can review and reason about. Beads core should own the generic
backend plugin process boundary. DoltLite-specific storage should live in an
external plugin repo. Gas City and the beads-doltlite packs should wire that
plugin into init, install, metadata, and runtime usage.

## Main Repositories and Workspaces

### Beads Core PR

- Repo: `gastownhall/beads`
- Local workspace: `/data/projects/doltlite-gascity/workspaces/beads-plugin-architecture`
- Branch/bookmark: `feat/backend-plugin-architecture`
- PR: https://github.com/gastownhall/beads/pull/4561
- Current local change: `zplpzmvy` / `3a06075d` / `Add backend plugin process architecture`

This PR adds the core plugin process architecture:

- `backend/plugin/protocol.go`: public v1alpha1 protocol types.
- `internal/backend/provider.go`: backend provider registry.
- `internal/backend/pluginprocess/*`: client, protocol aliases, process adapter, and store implementation.
- `cmd/bd/store_factory.go`: opens `pluginprocess` when `.beads/metadata.json` has `backend_plugin_command`.
- `internal/configfile/configfile.go`: persists `backend_plugin_command` and `backend_plugin_args`.
- `cmd/bd/backend.go`: new draft `bd backend install <backend>` command started today.

The PR description has been updated to be maintainer-facing and self-contained.
It links only to the external plugin repo, not to local-only paths.

Current caveat: `cmd/bd/backend.go` and `cmd/bd/backend_test.go` are new local
work after the PR body update. Focused `internal/configfile` and
`internal/backend/pluginprocess` tests passed. A targeted `cmd/bd` package test
and a plain `go build ./cmd/bd` were stopped because they behaved like broad,
slow `cmd/bd` checks.

### DoltLite Backend Plugin

- Repo: `duncan4123/beads-backend-doltlite`
- Local workspace: `/data/projects/doltlite-gascity/rigs/beads-backend-doltlite-plugin`
- Main commit pushed: `340942f1d016` / `Document DoltLite backend plugin architecture`

This repo is the concrete external plugin proof for Beads PR #4561.

Important pieces:

- `cmd/bd-backend-doltlite`: plugin binary.
- `backend/plugin/protocol.go`: local copy of the protocol types while the SDK boundary is still in review.
- `internal/provider`: session manager and DoltLite provider.
- `internal/storage`: copied DoltLite-backed Beads storage implementation.
- `scripts/build.sh`: builds `bd-backend-doltlite` against a DoltLite shared library.
- `README.md`: updated today to describe the plugin architecture, build, metadata config, tracing, implemented surface, smoke test, conformance direction, and open design questions.

Runtime shape:

```text
bd command
  -> Beads backend provider registry
  -> plugin-process storage adapter
  -> bd-backend-doltlite over stdio
  -> DoltLite-backed storage
```

### Beads DoltLite Pack Branch

- Repo: `duncan4123/gascity-packs`
- Clean branch workspace: `/data/projects/doltlite-gascity/gascity-packs-land-beads-doltlite`
- Bookmark: `feature/beads-doltlite-backend-plugin`
- Current change: `yusrqkml` / `4a243969` / `Wire beads-doltlite pack to backend plugin`

This is the clean pack branch for the plugin-aware DoltLite pack work. Use this
workspace for branch/PR work, not the default `gascity-packs` workspace.

Important changes:

- `beads-doltlite/assets/scripts/gc-beads-doltlite-bd.sh`
  - Writes plugin-aware `.beads/metadata.json`.
  - Supports `GC_DOLTLITE_BACKEND_PLUGIN=off`.
  - Supports explicit `GC_DOLTLITE_BACKEND_PLUGIN_COMMAND`.
  - Finds/copies `bd-backend-doltlite` into pack runtime state when available.
  - Writes `backend_plugin_command` and trace-enabled `backend_plugin_args`.
- `beads-doltlite/commands/build/run.sh`
  - Adds `plugin` build target.
  - Builds `./cmd/bd-backend-doltlite` from the plugin repo.
  - Installs to `.gc/runtime/packs/beads-doltlite/bin/bd-backend-doltlite`.
  - Writes `last-build-bd-backend-doltlite.json`.
- `beads-doltlite-init`
  - Now contains the fuller init pack with `assets/scripts/gc-beads-doltlite-bd.sh`.
  - The init script writes plugin metadata when a plugin binary already exists, but still bootstraps plain DoltLite when no plugin is installed.

Verified today:

- `bash -n` passed for all patched init scripts.
- Script smoke with fake `bd` showed:
  - no plugin present: metadata stays DoltLite-only.
  - plugin present: metadata includes `backend_plugin_command` and `backend_plugin_args`.

### Default Gascity-Packs Workspace

- Path: `/data/projects/doltlite-gascity/gascity-packs`
- Current change: `psomunkv` / `168af5ca` / `review: Re-review fixed JJ work`

This workspace is mixed and should not be used for clean DoltLite pack release
work until it is split or reset intentionally.

It currently contains:

- unrelated `gascity-jj-base` review edits that predated this work.
- copied `beads-doltlite-init` fuller pack.
- added DoltLite pack entries in `registry.toml`.

Use it only as the "default workspace has the init pack available" working copy
until the unrelated edits are separated.

### Gas City Core

- Repo/workspace: `/data/projects/doltlite-gascity/gascity`
- Current change: `mqpzpupt` / `afb051ef` / `document: re-review jj fixed work`
- Relevant file changed today:
  - `internal/bootstrap/packs/beadsdoltliteinit/assets/scripts/gc-beads-doltlite-bd.sh`

This embedded init pack is the one that matters before external packs are
available. The external `beads-doltlite-init` pack in gascity-packs is useful as
a dereferenceable public pack, but a fresh `gc init --beads-backend doltlite`
uses the embedded copy from the `gc` binary during bootstrap.

The embedded init script now:

- runs `bd init --backend doltlite` for bootstrap.
- rewrites metadata after bootstrap.
- includes plugin metadata only if a plugin executable is already present.
- does not fail fresh init just because the plugin has not been installed yet.

There is a `main` bookmark conflict in this repo that should be resolved before
release work.

### Root City Configuration

- City root: `/data/projects/doltlite-gascity`
- Root config changed today:
  - `city.toml`: `tick_debounce` changed from `5s` to `30s`.
  - `.gc/site.toml`: `gascity-packs` rig path changed to the branch workspace
    `/data/projects/doltlite-gascity/gascity-packs-land-beads-doltlite` so the
    city can load the plugin-aware `beads-doltlite` pack.

Current root workspace has other existing planning changes:

- `city.toml`
- `pack.toml`
- `packs.lock`
- `plans/order-system-recovery-20260701/*`

Do not treat those as part of the backend plugin change without re-checking.

## How the Pieces Work Together

1. Gas City selects DoltLite with:

   ```toml
   [beads]
   backend = "doltlite"
   bd_compatibility = "bd-1.0.5"
   ```

2. `gc init` needs a bootstrap provider before external packs are installed.
   That is the embedded `beads-doltlite-init` pack in the `gc` binary.

3. The init provider creates the DoltLite Beads store with `bd init --backend doltlite`.

4. The init provider writes `.beads/metadata.json`.
   - If no plugin binary exists, metadata stays plain DoltLite.
   - If a plugin binary exists, metadata includes:

     ```json
     {
       "backend": "doltlite",
       "database": "doltlite",
       "dolt_database": "hq",
       "backend_plugin_command": "/path/to/bd-backend-doltlite",
       "backend_plugin_args": ["--trace", "/path/to/backend-plugin-trace.jsonl", "serve"]
     }
     ```

5. The operational `beads-doltlite` pack can build and install the plugin:

   ```bash
   gc beads-doltlite build plugin --install --no-restart \
     --plugin-source /path/to/beads-backend-doltlite-plugin \
     --lib /path/to/doltlite/lib
   ```

6. Once metadata has `backend_plugin_command`, Beads core opens storage through
   `internal/backend/pluginprocess` instead of a built-in provider.

7. The DoltLite plugin process receives protocol requests over stdio, opens the
   real DoltLite store, and serves Beads operations.

## New `bd backend install` Command

The first draft was started in Beads core:

```bash
bd backend install doltlite \
  --command /path/to/bd-backend-doltlite \
  --trace /path/to/backend-plugin-trace.jsonl
```

Intended behavior:

- validate the plugin executable.
- preserve existing metadata fields like `dolt_database` and `project_id`.
- write `backend`, `database`, `backend_plugin_command`, and `backend_plugin_args`.
- append `serve` if not already present.
- run without opening the Beads store.

This command should become the generic Beads-side installer for already-built
backend plugin binaries. It does not build plugin binaries; pack commands still
own build/install of a particular plugin implementation.

## Gas City Formula Context

The `gascity` pack README describes the workflow system we will likely use for
review/release work:

- `gc.mayor` is the coordinator skill. It inspects, writes requirements/plans,
  creates approved beads/convoys, and launches formulas.
- `build-basic` is the full lifecycle formula: requirements, implementation
  plan, design review, task decomposition, implementation drain, review,
  finalize, publish.
- `build-from-*` formulas are continuation entrypoints when earlier artifacts
  already exist.
- `implement` drains an approved implementation convoy directly.
- `github-pr-review`, `github-issue-triage`, and `github-issue-fix` are
  targetless GitHub adapters.
- `build-base` is the virtual contract. Methodology packs extend it, but users
  launch cataloged concrete formulas.

For release preparation, the likely workflow shape is:

- Use Mayor to create a short release-readiness plan if we need structured beads.
- Use `build-from-review` or `github-pr-review` for PR review/reporting when
  implementation is already mostly done.
- Use direct `implement` only if we create a convoy of concrete cleanup tasks.

## Release Readiness Checklist

Beads core:

- Finish or remove `bd backend install` depending on whether it should be part
  of PR #4561.
- Re-run focused tests:
  - `go test ./backend/plugin ./internal/backend/pluginprocess ./internal/configfile`
  - avoid broad `go test ./cmd/bd`; use a narrow command/build check only when
    we accept the runtime cost.
- Update PR #4561 if the command is included.

DoltLite plugin repo:

- Confirm `main` is clean and pushed.
- Rebuild against the intended DoltLite library.
- Smoke through `BD_BIN=/path/to/bd ./scripts/smoke-core-adapter.sh`.
- Consider tagging an `rc1` only after the core adapter and metadata install
  path are tested together.

gascity-packs:

- Use `/data/projects/doltlite-gascity/gascity-packs-land-beads-doltlite` for
  the clean plugin-aware pack branch.
- Separate or ignore the mixed default workspace until the `gascity-jj-base`
  edits are accounted for.
- Decide whether `beads-doltlite-init` fuller pack and registry entries belong
  in the same pack PR or a separate PR.
- Push the latest init-pack updates after review.

gascity:

- Rebuild `gc` after the embedded init script change.
- Run a clean, bounded `gc init --beads-backend doltlite` smoke.
- Verify metadata with and without a preinstalled plugin binary.
- Resolve the `main` bookmark conflict before release.

Root city:

- Keep `tick_debounce = "30s"` unless supervisor responsiveness becomes a problem.
- Keep `gascity-packs` rig pointed at the plugin-aware branch workspace while
  testing this city.

## Current Known Risks

- The command/test surface in `cmd/bd` is expensive; avoid accidental full
  package tests.
- Default `gascity-packs` is mixed with unrelated review work.
- `gascity` has a bookmark conflict on `main`.
- Plugin raw SQL support was discussed and `bd sql` previously returned
  `storage backend does not support raw DB access`; verify whether the current
  branch includes the intended fix before claiming SQL is complete.
- The init path now writes plugin metadata opportunistically, but a generic
  "build plugin then enable metadata everywhere" command is still incomplete.

## Skill Candidate

A future skill for this work should teach:

- Read Beads PR #4561 first for the core plugin boundary.
- Read `beads-backend-doltlite/README.md` for the external plugin shape.
- Treat `.beads/metadata.json` as the switch that makes Beads use the plugin:
  `backend_plugin_command` plus `backend_plugin_args`.
- Build/install plugin binaries through the `beads-doltlite` pack, not generic
  Beads core.
- Use `bd backend install` once it lands to write metadata for an already-built
  plugin.
- For Gas City init, remember the embedded `beads-doltlite-init` pack in the
  `gc` binary is the pre-external-pack bootstrap surface.
- Always check jj workspace hygiene before release: Beads core PR workspace,
  plugin repo main, clean gascity-packs branch workspace, gascity embedded init
  workspace, and root city config.
