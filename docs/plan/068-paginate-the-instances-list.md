# [instances]: paginate the instances list

- **Plan id:** issue 68
- **Area:** Instances
- **Status:** completed 2026-10-09
- **Issue:** [#68](https://github.com/dimasbaguspm/noxis/issues/68)
- **Pull request:** [#75](https://github.com/dimasbaguspm/noxis/pull/75)
- **Commit:** [61a7073](https://github.com/dimasbaguspm/noxis/commit/61a707320ddd6f2b1d7656b85812e2995014ad0d) — `[ai:y][#68] feat: paginate the instances list (#75)`

## Plan

## What's problem?

The instances list can grow without bound and the dashboard loads it all at once; there is no way to page through many instances.

## What's affected and how?

- The instances endpoint accepts `page` and `perPage` and returns `total` alongside the list.
- The dashboard shows shadcn pagination, a skeleton while loading, and the shared empty state when there are no rows.

## Definition of Done

- The dashboard pages through instances.
- The name/driver filters keep working with pagination.
- The API reports the true total for the filtered result.

## Additional Notes

- none

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
