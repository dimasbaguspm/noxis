# [console][web] tile the DML/DDL search page like the instance page

- **Plan id:** issue 147
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#147](https://github.com/dimasbaguspm/noxis/issues/147)
- **Pull request:** [#157](https://github.com/dimasbaguspm/noxis/pull/157)
- **Commit:** [6ce6e3d](https://github.com/dimasbaguspm/noxis/commit/6ce6e3d24ee0106cdf3eecadcbda09daf32688f1) — `[ai:y][#147] refactor: tile the DML/DDL search page like the instance page (#157)`

## Plan

## What's problem?
The DML/DDL page keeps filters inline with the create CTA; it should match the new tile layout.

## What's affected and how?
- change-page.tsx: split into a filter card (search + status), a right-aligned CTA row (create trigger), and the results card

## Definition of Done
DML and DDL render as <Filter tile> <cta right aligned> <Search result tile>.

## Additional Notes

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
