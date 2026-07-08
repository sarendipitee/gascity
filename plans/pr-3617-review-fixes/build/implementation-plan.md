---
schema: gc.build.plan.v1
workflow:
  id: gc-4n3b
  formula: build-basic
methodology:
  pack: gascity
  name: build-basic
producer:
  formula: build-basic
  stage: plan
  attempt: 1
status: approved
trace:
  upstream:
    - path: /data/projects/doltlite-gascity/gascity/plans/pr-3617-review-fixes/review-input.md
      hash: sha256:e48e76c0191d2a728649141ebe44be22a63f12dd1d7596d1fde909524046dbba
      ids:
        - REQ-001
        - REQ-002
    - path: /data/projects/doltlite-gascity/gascity/plans/pr-3617-review-fixes/context.yaml
      hash: sha256:f2abad33266d976ec66e471d36343df6906950b398b24b88f91ead7571aeff03
    - path: /data/projects/doltlite-gascity/gascity/plans/pr-3617-review-fixes/build/code-review-context.md
      hash: sha256:aa97bae199eba3af17cec1d292c95ccff17e21e19a9c7658304f1f56bf33d69b
  coverage:
    - id: REQ-001
      status: covered
    - id: REQ-002
      status: covered
---

# Implementation Plan: PR 3617 Review Fixes

| ID | Status |
| --- | --- |
| REQ-001 | covered |
| REQ-002 | covered |

## Summary

Implement the requested PR 3617 review fixes by keeping generic Gas City core
environment variables pack-agnostic and either giving `gc.pack_root` a real
runtime consumer or removing it from the metadata surface.

The recorded generated requirements artifact at
`/data/projects/doltlite-gascity/gascity/plans/pr-3617-review-fixes/build/requirements.md`
was missing when this plan was written. This plan uses the recorded review input
as the requirements source because it contains the accepted `REQ-001` and
`REQ-002` acceptance criteria.

## Current System

The review input reports that generic core currently exposes pack routing as
`GC_PACKER_PACK` from the generic `gc.pack` metadata key. That leaks a
pack-specific name into the generic core environment surface.

The review input also reports that `internal/beadmeta/keys.go` added
`PackRootMetadataKey` for `gc.pack_root`, but the metadata key has no non-test
Go consumer. Leaving the key registered without runtime behavior makes the
metadata contract misleading.

## Proposed Implementation

Update the generic pool/session environment construction path so `gc.pack`
emits `GC_PACK`. When `gc.pack_root` is present and intentionally retained,
emit `GC_PACK_ROOT` from generic core as the pack root environment variable.
Remove any generic-core emission of `GC_PACKER_PACK`.

If packer compatibility still needs `GC_PACKER_PACK`, add it only in the
packer-owned pre-start layer by aliasing `GC_PACK` to `GC_PACKER_PACK`. Keep the
alias out of generic core so other packs do not inherit packer-specific names.

Resolve `gc.pack_root` deliberately. Prefer consuming it in the runtime path
that prepares pack-aware sessions by passing its value through `GC_PACK_ROOT`.
If implementation inspection shows no runtime should consume it, remove
`PackRootMetadataKey`, related producers, and tests that only preserve unused
scaffolding.

Keep the pool-agent predicate cleanup as a follow-up unless it is directly
touched while changing the environment construction path. The review note is
non-blocking and should not expand the core acceptance scope.

## Non-Goals

Do not redesign pack routing, change the build-basic workflow, or review source
quality without a recorded source anchor/worktree. Do not add a new metadata
primitive beyond the existing `gc.pack` and optional `gc.pack_root` keys.

## Verification

Run focused tests around pool desired-state trigger environment construction,
pack workspace setup, and metadata key registration. The tests must prove that
generic core emits `GC_PACK`, emits `GC_PACK_ROOT` when `gc.pack_root` is
retained, and does not emit `GC_PACKER_PACK`.

Run the relevant package tests for files touched in `internal/beadmeta`,
session/pool desired-state handling, and packer pre-start compatibility if that
alias is implemented. Run `go vet ./...` before landing.
