# [console][web] drop the redundant grid on connection step two

- **Plan id:** issue 143
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#143](https://github.com/dimasbaguspm/noxis/issues/143)
- **Pull request:** [#153](https://github.com/dimasbaguspm/noxis/pull/153)
- **Commit:** [2c3ef7d](https://github.com/dimasbaguspm/noxis/commit/2c3ef7d26aba57f357875d29c74190e5acb48c67) — `[ai:y][#143] refactor: drop the redundant grid on connection step two (#153)`

## Plan

## What's problem?
SheetBody carries 'grid gap-4' while the form inside already defines its own grid gap-4; the layout is doubled.

## What's affected and how?
- connection-dialog.tsx: remove className='grid gap-4' from the step-2 SheetBody

## Definition of Done
Step two renders with a single grid; visual spacing unchanged.

## Additional Notes
Keep the form's own grid gap-4.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
