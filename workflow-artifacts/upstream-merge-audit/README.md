# Upstream Merge Audit Artifacts

This directory captures parent-relative data for the temporary jj merge commit that joins the DoltLite fork tip with upstream `main`.

## Revisions

See `revisions.txt` for exact change IDs/commit IDs. The important anchors are:

- `@`: temporary integration merge commit.
- `ksxuolmk`: our DoltLite parent/tip before the merge.
- `qqoynwux`: upstream `main` parent.
- `tzwxltmy`: common base used for comparison.

## Files

- `status.txt`: current jj status and unresolved conflicts.
- `conflict-files.txt`: files still structurally conflicted in the merge.
- `from-our-tip.summary.txt`: all paths changed by the merge relative to our DoltLite tip. This is the primary "what upstream brought in" file list.
- `from-our-tip.patch`: full patch for the merge relative to our DoltLite tip.
- `cleanly-merged-files.txt`: paths from `from-our-tip.summary.txt` excluding unresolved conflict files.
- `focused-clean-merge.patch`: high-signal cleanly merged upstream changes for bd scope, graph routing, ready projection, and runtime overlay staging. Start here.
- `from-upstream-main.summary.txt` / `from-upstream-main.patch`: what our merged tree differs by relative to upstream `main`. This is mostly the DoltLite fork delta, not the primary upstream-port view.
- `upstream-vs-base.summary.txt`: upstream-only path list since the common base.
- `ours-vs-base.summary.txt`: our fork path list since the common base.

## Suggested Review Order

1. Read `focused-clean-merge.patch`.
2. Resolve the five files in `conflict-files.txt` by combining upstream pool/read hardening with DoltLite fork metadata/workspace behavior.
3. Use `from-our-tip.patch` only when broader context is needed.
