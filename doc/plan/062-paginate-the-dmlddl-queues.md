# [changes]: paginate the DML/DDL queues

- **Plan id:** issue 62
- **Area:** Changes & query
- **Status:** completed 2026-10-09
- **Issue:** [#62](https://github.com/dimasbaguspm/noxis/issues/62)
- **Pull request:** [#74](https://github.com/dimasbaguspm/noxis/pull/74)
- **Commit:** [fdad09d](https://github.com/dimasbaguspm/noxis/commit/fdad09df223fa8d834c2a3b8cea3b91dbb8b0a25) — `[ai:y][#62] feat: paginate the DML/DDL queues (#74)`

## Plan

## What's problem?

The DML and DDL queues load every change at once; with many changes there is no way to page through them.

## What's affected and how?

- The changes endpoint accepts `page` and `perPage` and returns `total`.
- Both queues show shadcn pagination, a skeleton while loading, and the empty state.

## Definition of Done

- The DML and DDL queues page through changes.
- The instance-name filter keeps working with pagination.
- The API reports the true total for the filtered queue.

## Additional Notes

- Blocked by the current batch (#57/#58); it touches the same queue pages.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
