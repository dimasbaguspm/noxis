# [query]: free the editor row and kill the stray scrollbars

- **Plan id:** issue 65
- **Area:** Changes & query
- **Status:** completed 2026-10-09
- **Issue:** [#65](https://github.com/dimasbaguspm/noxis/issues/65)
- **Pull request:** [#69](https://github.com/dimasbaguspm/noxis/pull/69)
- **Commit:** [31a9856](https://github.com/dimasbaguspm/noxis/commit/31a98569dacf4db3765e6b14e967fc376def0415) — `[ai:y][#65] fix: free the query editor row and fix panel overflow (#69)`

## Plan

## What's problem?

The query page spends its top row on a statement-kind badge and a help tooltip, and the panels show stray horizontal and vertical scrollbars even when there is nothing to scroll.

## What's affected and how?

- The kind badge and the tooltip are removed so the resizable panes span the whole width.
- The vertical panel group uses `pl-2` instead of `ml-2`, and the panels are spaced with a gap, so margins no longer force overflow scrollbars.

## Definition of Done

- The query page shows no kind badge and no tooltip.
- No horizontal or vertical scrollbar appears in the query panes when they are empty.
- The spacing between the editor and result panels looks intentional.

## Additional Notes

- none

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
