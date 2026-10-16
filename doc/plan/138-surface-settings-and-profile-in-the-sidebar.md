# [console][web] surface Settings and Profile in the sidebar

- **Plan id:** issue 138
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#138](https://github.com/dimasbaguspm/noxis/issues/138)
- **Pull request:** [#148](https://github.com/dimasbaguspm/noxis/pull/148)
- **Commit:** [c2e701a](https://github.com/dimasbaguspm/noxis/commit/c2e701a8a182eb0f9a5e491146ceb2226ba0a012) — `[ai:y][#138] feat: surface Settings and Profile in the sidebar (#148)`

## Plan

## What's problem?
Settings and Profile live hidden (profile only in the NavUser dropdown, access labelled 'Access'), so nobody finds them.

## What's affected and how?
- app-sidebar.tsx: add an Account group with Settings and Profile items; drop the 'Access' item from the Console group

## Definition of Done
A Settings item and a Profile item are visible in the sidebar and navigate to /settings and /profile.

## Additional Notes
Route constants PAGE_ROUTES.SETTINGS / PROFILE exist.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
