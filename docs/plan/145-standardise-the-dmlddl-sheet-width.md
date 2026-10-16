# [console][web] standardise the DML/DDL sheet width

- **Plan id:** issue 145
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#145](https://github.com/dimasbaguspm/noxis/issues/145)
- **Pull request:** [#155](https://github.com/dimasbaguspm/noxis/pull/155)
- **Commit:** [5554f78](https://github.com/dimasbaguspm/noxis/commit/5554f7821c128fea6a2f259eca156891623cb2f1) — `[ai:y][#145] refactor: standardise the DML/DDL sheet width (#155)`

## Plan

## What's problem?
The DML/DDL sheets are wider than every other sheet (max-w-2xl / max-w-3xl vs the standard max-w-md).

## What's affected and how?
- create-change-sheet.tsx and change-submit-dialog.tsx: drop the width overrides so both use the standard sheet width

## Definition of Done
Saving a DML/DDL change and reviewing one both open at the standard sheet width.

## Additional Notes
Standard = the Sheet default (max-w-md).

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
