# [console][web] label the filter fields on Instances, DML and DDL

- **Plan id:** issue 161
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#161](https://github.com/dimasbaguspm/noxis/issues/161)
- **Pull request:** [#163](https://github.com/dimasbaguspm/noxis/pull/163)
- **Commit:** [27c1781](https://github.com/dimasbaguspm/noxis/commit/27c1781c1f7a66b00106388c5ce76a29ca083c3d) — `[ai:y][#161] refactor(web): label and stack the filter fields (#163)`

## Plan

## What's problem?
The filter rows on the Instances page and the DML/DDL change page are unlabeled controls: a search input with placeholder text and a select, side by side, with no field label.

## What's affected and how?
web/src/pages/dashboard.tsx (Instances filter) and web/src/pages/change-page.tsx (DML/DDL filter, shared component). Users cannot tell at a glance what each control filters (name vs driver/status).

## Definition of Done
- Both filter cards show a Label above each control (Search by name / Driver / Status).
- Controls are stacked one per row instead of side by side.
- typecheck/lint/build pass.

## Additional Notes
DML and DDL share change-page.tsx, so one change covers both.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
