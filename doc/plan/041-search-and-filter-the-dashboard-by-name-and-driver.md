# [instances]: search and filter the dashboard by name and driver

- **Plan id:** issue 41
- **Area:** Instances
- **Status:** completed 2026-10-09
- **Issue:** [#41](https://github.com/dimasbaguspm/noxis/issues/41)
- **Pull request:** [#45](https://github.com/dimasbaguspm/noxis/pull/45)
- **Commit:** [562342b](https://github.com/dimasbaguspm/noxis/commit/562342bf3b5069dfeb3e7ebd38d454fb0269a5f6) — `[ai:y][#41] feat: search and filter the dashboard by name and driver (#45)`

## Plan

## What's problem?

The dashboard pulls three family lists and filters them in the browser, and the driver choices are duplicated between the UI and the server. There is no shared, server-driven way to search instances by name or driver.

## What's affected and how?

- New cross-family instance list with optional name, driver and host filters.
- New hardcoded driver catalog endpoint returning driver, family and label.
- The console loads its instance registry from the list endpoint instead of three separate calls.
- The dashboard drops the stat tiles (instances, DDL/DML executed, error rate) and shows a name / driver / detail table with a name search box and a driver dropdown fed by the catalog.

## Definition of Done

- The dashboard shows name, driver and a Detail action; the stat tiles are gone.
- Filtering by name and by driver both narrow the table.
- The driver dropdown is populated from the server catalog, not a UI constant.
- A caller only sees instances they can read.

## Additional Notes

- Recommend landing after #40 to avoid churning the same dashboard surface.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
