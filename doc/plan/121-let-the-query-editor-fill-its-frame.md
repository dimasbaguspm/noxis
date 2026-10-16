# [console]: let the query editor fill its frame

- **Plan id:** issue 121
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#121](https://github.com/dimasbaguspm/noxis/issues/121)
- **Pull request:** [#130](https://github.com/dimasbaguspm/noxis/pull/130)
- **Commit:** [333b763](https://github.com/dimasbaguspm/noxis/commit/333b7635ab608b2273ab368792ceb9ae86f82bf2) — `[ai:y][#121] refactor(web): let the query editor fill its frame (#130)`

## Plan

## What's problem?

The query page hard-codes a calc height (100svh minus a header allowance), which breaks as soon as the shell chrome changes.

## What's affected and how?

- The query page root switches to h-full and lets the shell's flex column define its height.
- The editor, schema panel and result grid keep their internal resizable layout.

## Definition of Done

- The query page fills its frame without a calc; panels still resize and scroll on their own.

## Additional Notes

- Blocked by nothing.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
