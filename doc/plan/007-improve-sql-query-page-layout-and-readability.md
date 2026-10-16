# [web]: improve SQL Query page layout and readability

- **Plan id:** issue 7
- **Area:** Console (web)
- **Status:** completed 2026-10-07
- **Issue:** [#7](https://github.com/dimasbaguspm/noxis/issues/7)
- **Pull request:** [#8](https://github.com/dimasbaguspm/noxis/pull/8)
- **Commit:** [b4f06dc](https://github.com/dimasbaguspm/noxis/commit/b4f06dcd8bd183992b93f7f11fcdb6b08ae41e4a) — `[ai:y][#7] feat: improve the SQL query page layout and readability (#8)`

## Plan

## What's problem?

The SQL Query page has rough edges that make it harder to read and use:

1. Labels and short strings render as raw text instead of title case.
2. The editor tab strip reopens whatever was left from a previous session instead of starting on Query 1.
3. With no SQL connection registered (or the active one unavailable) the page renders empty panes with no explanation.
4. The table browser's expandable row shows each column's SQL data type, which is noise when browsing.
5. The result grid overflows sideways instead of scrolling inside its card, so the whole page scrolls horizontally.
6. There is stray vertical space above the editor tab strip.
7. The table metadata is plain text and hard to scan.

## What's affected and how?

Console SQL Query page only. Query execution and the API are unchanged; this is presentation.

- Text uses `startCase` from `radash` (already a dependency).
- The tab strip initializes to a single tab named "Query 1" on every visit.
- A missing/unavailable connection shows a Not Found state with a way to register one, instead of empty panes.
- The expandable table row drops the data-type column.
- The result grid scrolls horizontally within its card; the page does not scroll sideways.
- Remove the gap above the tab strip.
- Table metadata is rendered in a scannable layout instead of a plain-text line.

## Definition of Done

- [ ] Labels on the page render in title case.
- [ ] Opening the Query page always starts with one tab named "Query 1".
- [ ] No connection registered, or the active one unavailable, shows a Not Found state with a link to add one; no empty editor/result shells.
- [ ] Expanding a table row no longer shows the SQL data type.
- [ ] The result grid scrolls horizontally inside its card; the page itself does not scroll sideways.
- [ ] No extra vertical space above the editor tab strip.
- [ ] Table metadata is scannable, not a plain-text line.
- [ ] `pnpm typecheck`, `pnpm lint`, `pnpm build` pass.

## Additional Notes

- `startCase` comes from radash (existing dependency).

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
