# Gas City Backend Plugin Surface

Gas City's backend plugin surface is defined by the operations `gc` performs
against bead storage without shelling through `bd`.

## Implemented

- Base `beads.Store` operations:
  - create/get/update/close/reopen/delete
  - list, ready, children, label, assignee, metadata filters
  - metadata slot set/batch set
  - dependency add/remove/list
  - transaction callback fallback through sequential plugin calls
- Optional store capabilities:
  - `CreateWithStorage`
  - `ReleaseIfCurrent`
  - `DepListBatch`
  - `Count`
  - `CloseStore`
- Backend process lifecycle:
  - metadata command discovery through `gascity_backend_command`
  - backwards-compatible DoltLite fastpath command aliases
  - `gascity.backend.v1alpha1` hello/capabilities/session protocol

## Still Direct Or Fallback

- `StorageGraphApplyStore.ApplyGraphPlanWithStorage`
  - This needs a first-class protocol method carrying `GraphApplyPlan`,
    `StorageClass`, and `GraphApplyResult`.
- Legacy linked `DoltliteReadStore`
  - Still present as fallback for linked builds.
  - Its read/write semantics should be treated as the compatibility oracle while
    moving additional helper methods into the backend protocol.
- DoltLite-specific health/maintenance helpers
  - These belong in backend admin capabilities rather than the base bead store
    protocol.

## Metadata

Preferred key:

```json
{
  "gascity_backend_command": "/absolute/path/to/gc-doltlite-fastpath",
  "gascity_backend_args": ["serve"]
}
```

Compatibility aliases remain supported while existing packs catch up:

- `gascity_fastpath_command`
- `gc_fastpath_plugin_command`
- `doltlite_fastpath_command`
- sibling `gc-doltlite-fastpath` inferred from `backend_plugin_command`
