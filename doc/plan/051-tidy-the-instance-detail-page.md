# [instances]: tidy the instance detail page

- **Plan id:** issue 51
- **Area:** Instances
- **Status:** completed 2026-10-09
- **Issue:** [#51](https://github.com/dimasbaguspm/noxis/issues/51)
- **Pull request:** [#59](https://github.com/dimasbaguspm/noxis/pull/59)
- **Commit:** [fcc2ba7](https://github.com/dimasbaguspm/noxis/commit/fcc2ba7a938ab1dca0f12835f68de6edbfd8b0e5) — `[ai:y][#51] refactor: tidy the instance detail page (#59)`

## Plan

## What's problem?

The breadcrumb repeats the instance id, and the page carries a changes table that duplicates the DML/DDL queues.

## What's affected and how?

- The instance breadcrumb reads Dashboard / Instances / Instance detail, with no repeated hash.
- The changes table is removed; the identity block and the test/delete/rotate actions stay.
- The now-unused instance-changes endpoint is removed.

## Definition of Done

- The breadcrumb has no duplicate id.
- The page shows only the common instance information and its actions.

## Additional Notes

- Keep the endpoint if we want the grouped changes later; otherwise remove it with the page section.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
