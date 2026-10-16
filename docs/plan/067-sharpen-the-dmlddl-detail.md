# [changes]: sharpen the DML/DDL detail

- **Plan id:** issue 67
- **Area:** Changes & query
- **Status:** completed 2026-10-09
- **Issue:** [#67](https://github.com/dimasbaguspm/noxis/issues/67)
- **Pull request:** [#72](https://github.com/dimasbaguspm/noxis/pull/72)
- **Commit:** [38f44c2](https://github.com/dimasbaguspm/noxis/commit/38f44c25148572b1c581a5ff91e8712c33601bc4) — `[ai:y][#67] feat: sharpen the change detail page (#72)`

## Plan

## What's problem?

The change detail breadcrumb ends in a raw id hash, the header shows only a bare status badge, and the schedule menu item reads "Schedule it…".

## What's affected and how?

- Breadcrumb for a change reads Dashboard / SQL / DML / DML Detail (and the DDL equivalent).
- Notable states show a descriptive Alert instead of only a badge (for example scheduled-for time, validation failure, applied).
- The Schedule control is a shadcn DropdownMenu with items "Immediate" and "Schedule it" (no ellipsis).

## Definition of Done

- The change detail breadcrumb never shows the id hash.
- The detail header surfaces notable states with an Alert.
- The Schedule menu shows exactly "Immediate" and "Schedule it".

## Additional Notes

- Blocked by the current batch (#57/#58); the schedule control exists in #50/#58.
- Ambiguity flagged: "more specific tile" interpreted as an Alert for notable states; adjust if another meaning was meant.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
