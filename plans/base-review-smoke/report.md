---
schema: gc.build.review.v1
workflow:
  id: gc-oegx
  formula: review
methodology:
  pack: gascity
  name: report
producer:
  formula: review
  stage: write-report
  attempt: 1
status: blocked
trace:
  upstream:
    - path: /data/projects/doltlite-gascity/gascity/plans/base-review-smoke/subject.md
      hash: missing:subject-not-found
      ids:
        - SUBJECT
  coverage:
    - id: SUBJECT
      status: blocked
      rationale: The declared subject file is missing, so there is no review target to inspect.
---

# Base Review Smoke Report

Generated: `2026-06-24T13:59:48Z`

## Verdict

Verdict: `blocked`

The review cannot pass or fail the subject because the declared subject file is missing.

| ID | Status |
| --- | --- |
| SUBJECT | blocked |

## Findings

### Blocking: declared subject file is missing

- Subject path: `/data/projects/doltlite-gascity/gascity/plans/base-review-smoke/subject.md`
- Report path: `/data/projects/doltlite-gascity/gascity/plans/base-review-smoke/report.md`
- Impact: there is no artifact, diff, or source description to review, so any pass/fail verdict would be fabricated.

Recommended fix: create the subject file with the intended review target, then rerun the review workflow.

## Verification

- Checked the declared subject path on disk: missing.
- Checked workflow root `gc-oegx` for the configured report and subject paths.
- Did not mutate code; this report records the missing evidence in report mode.
