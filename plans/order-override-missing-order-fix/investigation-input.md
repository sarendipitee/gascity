# Investigate And Fix Missing Order Override Warning

## Problem

Gas City supervisor/patrol logs repeatedly report:

```text
2026/06/24 07:32:19 gc patrol: order scan: order overrides: orders.overrides[4]: order "jjw-workspace-report" (rig "gascity") not found
```

The current city config contains disabled overrides for both `jjw-workspace-report`
and `workspace-report`, including rig-specific entries for `gascity`.

Relevant current config excerpt from `/data/projects/doltlite-gascity/city.toml`:

```toml
[[orders.overrides]]
name = "jjw-workspace-report"
enabled = false

[[orders.overrides]]
name = "workspace-report"
enabled = false

[[orders.overrides]]
name = "jjw-workspace-report"
rig = "gascity"
enabled = false

[[orders.overrides]]
name = "workspace-report"
rig = "gascity"
enabled = false
```

## Expected Behavior

Disabled order overrides should not produce repeated patrol scanner errors when
the referenced order is absent, unless the documented config contract explicitly
requires disabled overrides to name installed orders.

## Investigation Scope

- Determine whether the correct fix is data/config cleanup, order import repair,
  or core scanner behavior.
- If disabled overrides are intended to be valid tombstones, fix the scanner so
  disabled missing-order overrides are ignored or downgraded appropriately.
- If disabled overrides must always point at existing orders, update docs and
  create a config cleanup patch that removes the invalid override entries.
- Preserve behavior for enabled overrides: enabled overrides referencing missing
  orders should still surface a clear diagnostic.

## Acceptance Criteria

- The repeated `jjw-workspace-report` missing-order warning no longer appears
  for disabled override entries in this city.
- Enabled missing-order override diagnostics remain covered by tests.
- The chosen behavior is documented or already matches existing documentation.
- Relevant tests are added or updated around disabled order overrides.
