# DoltLite Front-Door Refactor Audit

Date: 2026-06-30

## Context

PR #3773 introduces the load-bearing seam: a compile-time,
per-coordination-class store boundary. On `main` every typed store still wraps
the same underlying store, but the point of the seam is that a future relocated
or per-class backend becomes a change at one resolution point instead of at
scattered call sites. The compiler should prevent handing a bead operation for
one class the store for another class.

PR #3800 builds on that seam by turning those typed stores into object-model
front doors: non-work domain objects must be accessed through typed APIs, with
bead serialization confined inside the front-door implementations.

PR #3777 gives the storage-side precedent for alternative ledger backends:
backend-specific logic belongs in lifecycle, environment projection, init, and
store-opening code.

The DoltLite integration should therefore plug into the #3773 class-store seam,
not remain a global side channel:

1. Resolve city/scope/class backend from config and metadata.
2. Open the class store at the single resolution point.
3. Run backend-specific lifecycle/init/env behavior from the same resolution
   model.
4. Construct the typed front door for that class.
5. Keep session/order/nudge/mail/control code backend-agnostic.

## Current Findings

The current workspace already contains the newer small-init-pack direction:

- `cmd/gc/beads_backend.go` defines `BeadsBackend`.
- `doltliteBackend.RequiredBuiltinPacks()` returns `beads-doltlite-init`.
- `internal/bootstrap/packs/beadsdoltliteinit/pack.toml` exists.
- `internal/builtinpacks/registry.go` registers `beads-doltlite-init`.
- `internal/config/public_packs.go` points the full `beads-doltlite` pack at
  a public pack source.
- `cmd/gc/embed_builtin_packs.go` adds the full `beads-doltlite` pack as an
  external import for DoltLite init, rather than bundling it directly.

That direction matches the user's stated product direction: Gas City embeds
only the minimal init support, while operational DoltLite commands live in
`gascity-packs`.

The pushed `pr3800-doltlite-parity-rebased` bookmark still contains a very
large historical delta, including docs, `.jjw` workspace metadata, plans,
schema snapshots, and the full `examples/beads-doltlite` pack. That branch
needs trimming before it becomes a clean integration PR.

## Boundary Assessment

### Keep

`BeadsBackend` is directionally right if it becomes the backend half of the
per-class store resolver, rather than a process-wide global switch. These
methods are appropriate at this layer:

- `Name`
- `NeedsManagedServer`
- `NeedsDoltBinary`
- `MinBDVersion`
- `NeedsBeadHooks`
- `NeedsDoltDoctorChecks`
- `MetadataInit`
- `MetadataEnforce`
- `ProviderEnv`
- `RequiredBuiltinPacks`

This mirrors PR #3777's split between managed-local Dolt and hosted gateway
behavior, while staying compatible with #3773's future per-class relocation
model.

### Refactor

Backend decisions are still spread across too many composition roots:

- `cmd/gc/main.go` opens optimized DoltLite stores directly.
- `cmd/gc/api_state.go` repeats optimized DoltLite store opening.
- `cmd/gc/bd_env.go` has scope-level DoltLite checks and env projection.
- `cmd/gc/beads_provider_lifecycle.go` mixes backend resolution with managed
  server lifecycle, metadata init, hooks, and readiness.
- `cmd/gc/embed_builtin_packs.go` knows both builtin init pack and external
  operational pack behavior.

The refactor should consolidate these into a class-aware store/backend resolver
that composition roots call once per coordination class.

### Avoid

Do not push backend branching upward into typed object code. Session, order,
nudge, mail, molecule, and reconciler paths should not ask whether the backing
ledger is DoltLite. If they need storage, they receive the class front door or
the class-specific store facade resolved for them.

Raw `beads.Store` remains legitimate for work beads and for composition roots,
but object-specific reads/writes should use:

- `session.InfoStore`
- `orders.Store` / order run front door
- `nudgequeue.Store`
- `mail.Provider`
- the work-assignment facade for work-bead assignment paths

## Proposed Refactor Shape

Introduce a narrow resolver around city/scope/class store construction,
roughly:

```go
type StoreClass string

const (
	StoreClassWork    StoreClass = "work"
	StoreClassSession StoreClass = "session"
	StoreClassOrder   StoreClass = "order"
	StoreClassNudge   StoreClass = "nudge"
	StoreClassMail    StoreClass = "mail"
)

type ScopeStoreResolver struct {
	CityPath string
	Config   *config.City
}

func (r ScopeStoreResolver) OpenStore(ctx context.Context, scopeRoot string, class StoreClass) (beads.Store, error)
func (r ScopeStoreResolver) Backend(scopeRoot string) BeadsBackend
func (r ScopeStoreResolver) BackendEnv(scopeRoot string) (map[string]string, error)
```

That resolver should be the local implementation of the #3773 seam. Today it
may return the same underlying store for every class, but the call shape must
preserve class identity so future per-class relocation is localized:

```go
store, err := resolver.OpenClassStore(ctx, scopeRoot, StoreClassSession)
if err != nil {
	return err
}
sessionInfo := session.NewInfoStore(store)
```

The exact type names should follow upstream #3773/#3800 naming if they land
with a `resolveClassStore` helper. The key ownership boundary is:

```text
resolve class store once -> construct class front door -> pass typed API down
```

Backend-specific decisions belong before or inside `resolveClassStore`, never
inside the domain call tree that consumes the front door.

## Implementation Beads

Suggested split:

1. **Class-store resolver cleanup**
   - Identify the upstream #3773 class-store resolution point.
   - Make DoltLite backend resolution feed that seam.
   - Preserve class identity even where all classes share the same store today.
   - Acceptance: focused class-store/front-door tests pass.

2. **Backend resolver cleanup**
   - Move duplicated optimized DoltLite open logic from `main.go` and
     `api_state.go` behind one resolver.
   - Keep hosted-gateway compatibility from #3777 in lifecycle/env paths.
   - Acceptance: focused store-opening and env tests pass.

3. **Init-pack cleanup**
   - Keep `beads-doltlite-init` builtin.
   - Keep full `beads-doltlite` as an external/public import.
   - Delete or update tests that expect the full pack to be bundled.
   - Acceptance: init/import tests pass for `bd + doltlite`.

4. **Front-door boundary audit**
   - Add or extend guard tests so fully converted files do not regain
     `beads.Store`.
   - Check DoltLite changes did not add backend-specific branches to typed
     object logic.
   - Acceptance: #3800 front-door guard passes.

5. **Branch hygiene**
   - Trim the parity branch to the real DoltLite integration delta.
   - Drop `.jjw`, temporary debug files, historical plans, generated snapshots,
     and full pack content that belongs in `gascity-packs`.
   - Acceptance: `jj diff --stat` is reviewable and scoped.

## Immediate Recommendation

Start with the #3773 class-store resolution point, then connect DoltLite backend
choice to it. Init-pack cleanup is the second step. Do not start by changing
session/order/nudge behavior; those should remain consumers of typed front
doors.
