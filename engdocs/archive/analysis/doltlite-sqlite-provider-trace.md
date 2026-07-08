# SQLite Coordstore Trace and DoltLite PR Implications

Date: 2026-06-06

## Question

Should DoltLite follow the same pattern as the upstream SQLite coordination
store, and does the current DoltLite work still belong behind the old
`[beads].backend` abstraction?

## Upstream SQLite Shape

Upstream has a production SQLite beads provider. It is selected as a normal
beads provider:

- `GC_BEADS=sqlite`
- `[beads] provider = "sqlite"`

The production open path is in `cmd/gc/main.go`:

- `rawBeadsProviderForScope(...)` resolves the configured provider.
- `providerIsCoordStore(...)` recognizes `sqlite`, plus deprecated aliases
  `sqlite-cgo` and `coordstore`.
- `openCoordStoreAt(...)` opens `beads.OpenSQLiteStore(...)`.
- The SQLite file lives under `scopeRoot/.gc/coordstore/beads.sqlite`.
- This branch returns before `beads.OpenStoreAtForCity(...)`, so it bypasses
  the bd/native-Dolt factory entirely.

The production implementation is `internal/beads/sqlite_store.go`. It
implements `beads.Store` directly: create/get/update/close/reopen/list/ready,
label/metadata queries, batch metadata, transaction wrapper, dependency APIs,
ping, close-store, retention, and busy retry handling.

The benchmark adapter at
`internal/benchmarks/coordstore/adapters/sqlite/adapter.go` is not the
production provider. It implements the separate `coordstore.StoreAdapter`
interface for benchmarking candidate coordination stores. The production
result of that work is `internal/beads/sqlite_store.go`.

## SQLite Coverage Pattern

SQLite has these upstream acceptance signals:

- Unit coverage in `internal/beads/sqlite_store_test.go`.
- Provider resolution and cross-store tests under `cmd/gc/`.
- Integration coverage through the `integration-sqlite-coordstore` CI job.
- CI sets both `GC_BEADS=sqlite` and
  `GC_ACCEPTANCE_BEADS_PROVIDER=sqlite`.
- The CI job runs `make test-integration-bdstore` and `make test-acceptance`.
- Workflow contract coverage in
  `.github/workflows/scripts/test_ci_suite_coverage.py`.
- Config docs list `sqlite` as a built-in beads provider and mark
  `sqlite-cgo`/`coordstore` as deprecated aliases.

The pattern is provider-first: one provider value owns selection,
store location, implementation, tests, docs, and CI.

## Current DoltLite Shape

DoltLite is not currently modeled like SQLite. It is modeled as a storage
backend under the bd-compatible provider:

- Provider selection still resolves to `bd` or `exec:gc-beads-bd`.
- `[beads].backend = "doltlite"` and `GC_BEADS_BACKEND=doltlite` select
  DoltLite behavior underneath the bd provider.
- `cmd/gc/providers.go` has `beadsBackend(...)` and
  `cityUsesDoltliteBeadsBackend(...)`.
- `cmd/gc/beads_provider_lifecycle.go` uses that backend flag to skip managed
  Dolt server lifecycle and hooks in some paths.
- `examples/bd/assets/scripts/gc-beads-bd.sh` contains the DoltLite init,
  metadata, maintenance, and bd CLI bridge behavior.
- `cmd/gc/cmd_bd_store_bridge.go` projects `GC_BEADS_BACKEND` and
  `BEADS_BACKEND` for bridge operations.
- With the `gascity_native_beads` build tag, `cmd/gc/doltlite_store_native.go`
  can wrap a `*beads.BdStore` with `beads.NewDoltliteReadStore(...)`. DoltLite
  scopes use this read path by default when the native-capable binary is built;
  `GC_NATIVE_DOLTLITE_BEADS=false` forces the bd CLI read path for parity
  debugging.
- `internal/beads/doltlite_read_store.go` is a read-optimized wrapper over a
  bd-backed DoltLite store. Writes still delegate to `BdStore`.

DoltLite therefore has two different concepts mixed together:

- A bd storage backend selected by `[beads].backend`.
- A native read fast path made available by build tag, defaulted by backend, and
  overrideable with `GC_NATIVE_DOLTLITE_BEADS`.

## Libdoltlite Build Finding

A Makefile-only build target with `CGO_ENABLED=1`, `-tags=libsqlite3`, and
`-ldoltlite` is not enough in current upstream. Validation showed the built
binary carried an rpath but no `NEEDED` entry for `libdoltlite`, and
`go list` did not show `modernc.org/sqlite` switching to a cgo/libsqlite3
driver path.

If Gas City must truly link to libdoltlite, it needs a real build-tagged
driver/store implementation that references the libdoltlite-backed SQLite ABI,
not just linker flags.

## Recommendation

Do not revive the old broad `BeadsBackend` abstraction as the primary upstream
story. Upstream has moved toward provider-level replacement for coordination
storage.

There are two viable paths:

1. Full DoltLite provider

   Follow the SQLite pattern. Add a provider-level DoltLite path, for example
   `provider = "doltlite"` if upstream agrees on the name. That path should:

   - Branch near `providerIsCoordStore(...)` / `openCoordStoreAt(...)`.
   - Open a DoltLite-backed implementation of `beads.Store` directly.
   - Own its storage directory and metadata.
   - Avoid bd CLI lifecycle for normal CRUD.
   - Have unit, provider-resolution, integration, acceptance, docs, and CI
     coverage analogous to SQLite.
   - If it depends on libdoltlite, make that dependency explicit through a real
     driver/store build tag and a verifiable linked binary test.

2. Narrow bd-backend compatibility fixes

   Keep DoltLite as `[beads].backend = "doltlite"` underneath the bd provider.
   In that case, keep changes narrow:

   - Fix `gc-beads-bd.sh` DoltLite init/maintenance bugs.
   - Fix doctor/lifecycle gates that still assume managed Dolt.
   - Keep `DoltliteReadStore` as a read-through optimization over `BdStore`.
   - Do not present this as a full replacement for Dolt/bd. Present it as
     compatibility support for bd's DoltLite backend.

For upstream PR readiness, the safest immediate stack is narrow bd-backend
fixes first. A full DoltLite provider should be a separate design/PR because it
should mirror the SQLite provider surface rather than layering more behavior
onto `[beads].backend`.
