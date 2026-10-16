# [console][web] split the instance page into filter, CTA and results tiles

- **Plan id:** issue 146
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#146](https://github.com/dimasbaguspm/noxis/issues/146)
- **Pull request:** [#156](https://github.com/dimasbaguspm/noxis/pull/156)
- **Commit:** [32bbfec](https://github.com/dimasbaguspm/noxis/commit/32bbfec6e9cda83bcebd7742ab6c1cb21d105528) — `[ai:y][#146] refactor: split the instance page into filter, CTA and results tiles (#156)`

## Plan

## What's problem?
The Instances page puts filters, title and 'New connection' in one card and the table inside it, mixing concerns.

## What's affected and how?
- dashboard.tsx: three stacked blocks — a filter card (search + driver), a right-aligned 'New connection' CTA row, and a results card (table + count + pagination)

## Definition of Done
The page renders as <Filter tile> <cta right aligned> <Search result tile> per the sketch.

## Additional Notes

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
