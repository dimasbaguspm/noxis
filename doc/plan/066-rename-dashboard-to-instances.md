# [console]: rename Dashboard to Instances

- **Plan id:** issue 66
- **Area:** Console (web)
- **Status:** completed 2026-10-09
- **Issue:** [#66](https://github.com/dimasbaguspm/noxis/issues/66)
- **Pull request:** [#71](https://github.com/dimasbaguspm/noxis/pull/71)
- **Commit:** [8315ab2](https://github.com/dimasbaguspm/noxis/commit/8315ab2a1e9116dfc5cf7485582640845a661b55) — `[ai:y][#66] refactor: rename the dashboard to Instances (#71)`

## Plan

## What's problem?

The console calls its top-level page "Dashboard", but the page is a list of instances, and everything downstream copies that label.

## What's affected and how?

- The sidebar entry, the page title, the breadcrumb root, and the instance back-link are renamed to "Instances".
- The route stays at the root; only labels change.

## Definition of Done

- No visible "Dashboard" label remains in the console.

## Additional Notes

- none

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
