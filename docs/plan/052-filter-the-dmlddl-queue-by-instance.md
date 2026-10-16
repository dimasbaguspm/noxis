# [changes]: filter the DML/DDL queue by instance

- **Plan id:** issue 52
- **Area:** Changes & query
- **Status:** completed 2026-10-09
- **Issue:** [#52](https://github.com/dimasbaguspm/noxis/issues/52)
- **Pull request:** [#56](https://github.com/dimasbaguspm/noxis/pull/56)
- **Commit:** [61f2a26](https://github.com/dimasbaguspm/noxis/commit/61f2a26f816db6713eb788e0169e06fe1cc1c99c) — `[ai:y][#52] feat: filter the DML/DDL queue by instance (#56)`

## Plan

## What's problem?

Both queues list every change, so finding one instance's changes means reading the whole list.

## What's affected and how?

- A name filter on the DML and DDL queues, like the dashboard's.
- The change list accepts a connection-name filter server-side, so the browser does not fetch the whole queue and trim it.

## Definition of Done

- Typing an instance name narrows the queue through the API; clearing it shows all.
- DML and DDL share the same filter.

## Additional Notes

- Match the dashboard filter's behavior so the surfaces stay consistent.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
