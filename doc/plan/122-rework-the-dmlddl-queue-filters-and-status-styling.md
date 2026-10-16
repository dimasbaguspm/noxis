# [console]: rework the DML/DDL queue filters and status styling

- **Plan id:** issue 122
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#122](https://github.com/dimasbaguspm/noxis/issues/122)
- **Pull request:** [#133](https://github.com/dimasbaguspm/noxis/pull/133)
- **Commit:** [814a5a2](https://github.com/dimasbaguspm/noxis/commit/814a5a257e9a012ce5fccde8b6261396337231b2) — `[ai:y][#122] feat(changes): filter the queue by status and style the rows (#133)`

## Plan

## What's problem?

The DML/DDL queue filters by instance only, its statuses render raw and colorless, rows mix case, the create button is long, and a subtitle claims a property (immediate and scheduled in one list) that is now just how the list works.

## What's affected and how?

- The create trigger is shortened to "Create".
- The filter bar matches the Instances page: search box + a single-select status filter (All statuses / unverified / validated / approved / applied / failed / rejected) backed by a new `status` query parameter on the changes list endpoint.
- Rows render title-cased values everywhere a value is shown.
- Status shows as a colored badge (success for applied, destructive for failed, neutral otherwise — Badge gains success/warning variants).
- The "immediate and scheduled in one list" subtitle is removed.

## Definition of Done

- Filtering by status returns only matching rows from the server (works across pagination).
- All visible row values are title-cased; status has a distinct color per state.
- The create button says "Create"; the subtitle is gone.

## Additional Notes

- Blocked by nothing; touches the Go list handler + UAT, and the badge component.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
