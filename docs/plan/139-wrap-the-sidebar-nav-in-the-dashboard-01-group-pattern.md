# [console][web] wrap the sidebar nav in the dashboard-01 group pattern

- **Plan id:** issue 139
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#139](https://github.com/dimasbaguspm/noxis/issues/139)
- **Pull request:** [#149](https://github.com/dimasbaguspm/noxis/pull/149)
- **Commit:** [389e951](https://github.com/dimasbaguspm/noxis/commit/389e951088c98bb28e8820c5cf6d8bf6728c54ff) — `[ai:y][#139] refactor: wrap the sidebar nav in the dashboard-01 group pattern (#149)`

## Plan

## What's problem?
Sidebar nav items sit flat in SidebarGroup; the dashboard-01 shell wraps them in SidebarGroupContent with a stacked menu.

## What's affected and how?
- app-sidebar.tsx: wrap the Console group menu and the Account group menu in SidebarGroupContent className='flex flex-col gap-2'

## Definition of Done
Each sidebar group is wrapped in the dashboard-01 structure per the reference snippet.

## Additional Notes

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
