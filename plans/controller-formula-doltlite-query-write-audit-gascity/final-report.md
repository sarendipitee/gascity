---
schema: gc.build.final-report.v1
workflow_root: gc-9d37
artifact_root: plans/controller-formula-doltlite-query-write-audit-gascity
generated_at: 2026-06-28T09:13:00Z
---

# Final Report: Controller Formula DoltLite Query/Write Audit

## Summary

The jj-build workflow for `gc-9d37` completed with `gc.outcome=pass` and was
configured to write managed documents under the default workspace artifact root:
`plans/controller-formula-doltlite-query-write-audit-gascity`.

This finalization pass found no existing manifest or intermediate document files
at that artifact root. The workflow root metadata records the intended
requirements path, document workspace, and manifest path, but the corresponding
files were absent when this report was generated.

## Outcome

- Workflow root: `gc-9d37`
- Input convoy: `gc-dip0`
- Document workspace: `default`
- Document base revset: `default@`
- Artifact root: `plans/controller-formula-doltlite-query-write-audit-gascity`
- Manifest: `plans/controller-formula-doltlite-query-write-audit-gascity/manifest.json`
- Build outcome metadata on the workflow root: `pass`

## Artifacts

- `final-report.md`: this managed final report.
- `manifest.json`: the managed document manifest updated with this report's
  path, schema, SHA-256 hash, and jj document change ID.

## Remaining Risks

- The expected upstream inputs for the build, including
  `requirements.md`, `implementation-plan.md`, decomposition output, and review
  output, were not present in the artifact root at finalization time.
- The configured validation script `.gc/scripts/checks/build-artifact-valid.sh`
  was not present in this checkout, so validation was limited to matching the
  established `gc.build.final-report.v1` frontmatter and manifest shape.
- Downstream consumers should treat this report as a finalization record for the
  current artifact state, not as evidence that the missing intermediate
  documents were produced.
