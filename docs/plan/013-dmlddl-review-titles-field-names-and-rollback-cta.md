# [web]: DML/DDL review titles, field names and rollback CTA

- **Plan id:** issue 13
- **Area:** Console (web)
- **Status:** completed 2026-10-08
- **Issue:** [#13](https://github.com/dimasbaguspm/noxis/issues/13)
- **Pull request:** [#14](https://github.com/dimasbaguspm/noxis/pull/14)
- **Commit:** [8613ffd](https://github.com/dimasbaguspm/noxis/commit/8613ffd8cad15508e009c7bef7fffaa9eb83e906) — `[ai:y][#13] feat: DML/DDL review titles, field names and rollback CTA (#14)`

## Plan

## What's problem?

The DML/DDL pages and review pages use lowercase labels, the review fields are named "Query"/"Rollback" (ambiguous), and there is no shortcut to raise the inverse change from one already reviewed.

## What's affected and how?

- Labels and short UI strings use radash's start case (`title`), consistent with the rest of the console.
- The create and review forms rename "Query" → "Original Query" and "Rollback" → "Rollback Query". UI-only; the API fields stay.
- The review page gets a CTA above "Rollback Query" that opens DML/DDL creation with the reviewed change's rollback query as the new "Original Query" — one click to raise the inverse change.

## Definition of Done

- [ ] DML/DDL create and review labels render in title case (radash `title`).
- [ ] The forms read "Original Query" and "Rollback Query".
- [ ] A CTA above "Rollback Query" on a reviewed change opens DML/DDL creation with the previous rollback prefilled as the original query.
- [ ] `pnpm typecheck`, `pnpm lint`, `pnpm build` pass.

## Additional Notes

- API/response field names stay `query`/`rollback`; the rename is UI-only.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
