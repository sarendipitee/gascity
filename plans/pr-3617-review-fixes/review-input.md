# PR 3617 Review Fix Input

Source: https://github.com/gastownhall/gascity/pull/3617
Repository: gastownhall/gascity
PR title: Route pack pool sessions from trigger beads
Head branch: duncan4123:trigger-bead-pack-workspaces
Head SHA: 540ce3457b9bdeb113da08f1dafbe8a6e1d74257
Base branch: gastownhall:main
Reviewer: sjarmak
Review state: CHANGES_REQUESTED
Review submitted: 2026-06-20T11:49:42Z

## Requested Changes

REQ-001: Keep generic core environment variables pack-agnostic.

The review says generic core currently exposes pack routing as `GC_PACKER_PACK`
from the generic `gc.pack` key. Replace the core-emitted environment surface
with generic `GC_PACK` and `GC_PACK_ROOT`. If packer compatibility is still
needed, alias from the generic variables inside the packer pre_start layer
instead of baking the packer name into generic core.

Acceptance:

- Generic Gas City core emits `GC_PACK` for `gc.pack`.
- Generic Gas City core emits `GC_PACK_ROOT` for `gc.pack_root` when present.
- `GC_PACKER_PACK` is not emitted by generic core.
- Packer scripts may preserve `GC_PACKER_PACK` only as a pack-local
  compatibility alias sourced from `GC_PACK`.
- Tests cover the generic env names and prevent pack-specific env names from
  reappearing in core.

REQ-002: Make `gc.pack_root` either consumed or removed.

The review says `internal/beadmeta/keys.go` added `PackRootMetadataKey` and
registered it as known metadata, but no non-test Go consumer exists. Either add
the intended Go consumer or remove the metadata key until it has a real
consumer.

Acceptance:

- If `gc.pack_root` remains in `internal/beadmeta/keys.go`, at least one
  non-test Go path consumes it for runtime behavior.
- If no runtime behavior should consume `gc.pack_root`, remove the key and
  any producer/tests that only support unused scaffolding.
- Tests verify the chosen behavior.

## Non-Blocking Review Notes

- Consider centralizing the duplicated pool-agent predicate instead of adding
  another local predicate.
- Check whether session-bead work directory metadata is written under both
  `work_dir` and `gc.work_dir`, and remove or document inert duplication.
- Add a rebind/clear-path test for the subtle pool-session rebinding path if it
  is still untested.

## Verification

Run targeted tests around pool desired-state trigger env, pack workspace setup,
metadata keys, and lint. Re-run broader command/package tests if the fix touches
shared session reconciliation or config validation.
