# [console][web] brand the sidebar header like dashboard-01

- **Plan id:** issue 140
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#140](https://github.com/dimasbaguspm/noxis/issues/140)
- **Pull request:** [#150](https://github.com/dimasbaguspm/noxis/pull/150)
- **Commit:** [11d9c5a](https://github.com/dimasbaguspm/noxis/commit/11d9c5a65206549853956e6cb22a785885e69ba4) — `[ai:y][#140] refactor: brand the sidebar header like dashboard-01 (#150)`

## Plan

## What's problem?
The site title doesn't match the dashboard-01 header pattern (single-line brand, size-5 icon, p-1.5 button padding).

## What's affected and how?
- app-sidebar.tsx: SiteTitle becomes a single-line 'noXis' brand + version, icon size-5, p-1.5 button per the reference.

## Definition of Done
SidebarHeader shows icon + noXis + version in one line; still links to /.

## Additional Notes

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
