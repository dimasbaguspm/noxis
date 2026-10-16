# [query]: fix the schema and result panel layout

- **Plan id:** issue 54
- **Area:** Changes & query
- **Status:** completed 2026-10-09
- **Issue:** [#54](https://github.com/dimasbaguspm/noxis/issues/54)
- **Pull request:** [#60](https://github.com/dimasbaguspm/noxis/pull/60)
- **Commit:** [9259069](https://github.com/dimasbaguspm/noxis/commit/9259069f83d755d9f7ab9b09f58deefc457e93dc) — `[ai:y][#54] fix: tidy the query page panels (#60)`

## Plan

## What's problem?

The schema panel is too wide, the right panel shows an empty horizontal scrollbar, the result panel shows an empty vertical scrollbar, and the database selector sits outside the resizable area.

## What's affected and how?

- The schema panel defaults narrower (still resizable).
- The stray overflow is removed so no empty horizontal or vertical scrollbars appear.
- The database/connection selector moves inside the left panel, above the schema tree.

## Definition of Done

- No empty horizontal scrollbar in the right panel and no empty vertical scrollbar in the result panel.
- The schema panel is narrower by default and still resizable.
- The connection selector sits above the schema tree inside the left panel.

## Additional Notes

- The top bar keeps the statement-kind badge.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
