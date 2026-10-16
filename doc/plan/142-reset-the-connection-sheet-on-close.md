# [console][web] reset the connection sheet on close

- **Plan id:** issue 142
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#142](https://github.com/dimasbaguspm/noxis/issues/142)
- **Pull request:** [#152](https://github.com/dimasbaguspm/noxis/pull/152)
- **Commit:** [8123313](https://github.com/dimasbaguspm/noxis/commit/81233133394c981484f9c65a0742cfc542cde022) — `[ai:y][#142] fix: reset the connection sheet on close (#152)`

## Plan

## What's problem?
Closing the connection sheet without saving keeps the old values; re-opening jumps to step two with stale fields instead of the kind selection.

## What's affected and how?
- connection-dialog.tsx: onOpenChange(false) resets the form to the family default, returns to step 1 and clears verified/busy

## Definition of Done
Reopening the sheet always shows the kind selection with blank form values.

## Additional Notes
Applies to both wizard usages (dashboard, query page).

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
