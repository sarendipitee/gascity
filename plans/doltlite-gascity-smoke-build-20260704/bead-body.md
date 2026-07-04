Smoke-test the `build-basic` formula path against recent gascity working-copy
changes related to DoltLite-backed beads and runtime behavior.

Use the subject artifact:
`/data/projects/doltlite-gascity/gascity/plans/doltlite-gascity-smoke-build-20260704/subject.md`

Primary goal:
- Run the basic build/review workflow far enough to validate that formula
  slinging, artifact creation, implementation review, and build checks can
  operate against the current jj working copy.

Review scope:
- DoltLite contract/file handling for city and rig stores.
- Dispatch/runtime behavior that touches DoltLite-backed beads.
- Template/environment resolution behavior affected by the current changes.

Operational constraints:
- Do not push.
- Do not open a PR.
- Treat this as a smoke build and review run, not a release workflow.
