# [console][web] make Instances a plain clickable item

- **Plan id:** issue 141
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#141](https://github.com/dimasbaguspm/noxis/issues/141)
- **Pull request:** [#151](https://github.com/dimasbaguspm/noxis/pull/151)
- **Commit:** [c781247](https://github.com/dimasbaguspm/noxis/commit/c781247044234653dad830f28e26d71f846f8db6) — `[ai:y][#141] refactor: make Instances a plain clickable item (#151)`

## Plan

## What's problem?
The Instances submenu (all instances + each registered one) adds a level nobody needs; the user wants it clickable without a submenu.

## What's affected and how?
- app-sidebar.tsx: drop InstancesMenu submenu + its usePagedInstancesQuery; Instances is a plain item linking to /

## Definition of Done
Instances renders as a single row without submenu and without fetching the instance list.

## Additional Notes
Sidebar stops re-querying instances.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
