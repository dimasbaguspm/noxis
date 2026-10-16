# [console]: fix the sheet structure: header, body, pinned footer

- **Plan id:** issue 94
- **Area:** Console (web)
- **Status:** completed 2026-10-09
- **Issue:** [#94](https://github.com/dimasbaguspm/noxis/issues/94)
- **Pull request:** [#101](https://github.com/dimasbaguspm/noxis/pull/101)
- **Commit:** [f6f7f19](https://github.com/dimasbaguspm/noxis/commit/f6f7f199273142850a5ba06e26a771d10406a5a4) — `[ai:y][#94] fix: pin the sheet footer below a scrollable body (#101)`

## Plan

## What's problem?

Sheet forms scroll: the action buttons ride inside the scrollable body, so on a tall form the user must scroll to the very bottom (past the fields) to Submit, and the footer is not visually anchored.

## What's affected and how?

All right-side panel forms get a predictable structure:

- Header area: sheet title + optional description, pinned top (the sheet gets a proper accessible title too).
- Scrollable body holding the form fields.
- Footer pinned below the body (buttons right-aligned, separated by a hairline) that never scrolls with the content.

## Definition of Done

- Every sheet renders Title, body and footer in that order; the footer stays visible while the body scrolls.
- The sheet announces a proper title to screen readers.
- No sheet still renders its action buttons inside the scrollable content.
- All existing sheets (connection, create DML/DDL, submit change, schedule, mongodb request) use the new structure with no visual regression.

## Additional Notes

- Blocked by nothing; start anytime.
- shadcn-style names: SheetHeader, SheetTitle, SheetDescription, SheetBody, SheetFooter.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
