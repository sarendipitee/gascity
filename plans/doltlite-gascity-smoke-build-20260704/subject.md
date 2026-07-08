# Smoke Build Subject: DoltLite usage in gascity

Review the recent working-copy changes in the gascity rig for how Gas City uses
DoltLite-backed beads and related runtime paths.

Repository:
- `/data/projects/doltlite-gascity/gascity`

Current jj working-copy summary:
- `cmd/gc/beads_provider_lifecycle.go`
- `cmd/gc/cmd_convoy_dispatch_test.go`
- `cmd/gc/dispatch_runtime.go`
- `cmd/gc/template_resolve.go`
- `cmd/gc/template_resolve_env_test.go`
- `internal/beads/contract/files.go`
- `internal/beads/contract/files_test.go`

Suggested inspection commands:
- `cd /data/projects/doltlite-gascity/gascity && jj st`
- `cd /data/projects/doltlite-gascity/gascity && jj diff --git`
- `cd /data/projects/doltlite-gascity/gascity && go test ./cmd/gc ./internal/beads/contract`

Review focus:
- DoltLite contract/file handling remains correct for city and rig stores.
- Runtime dispatch paths do not regress DoltLite-backed beads behavior.
- Template/env resolution changes do not break existing workflows.
- Tests cover the changed behavior or clearly identify missing coverage.

Constraints:
- Do not push or open a PR from this smoke run.
- Prefer report-only review unless a formula step explicitly owns safe fixes.
