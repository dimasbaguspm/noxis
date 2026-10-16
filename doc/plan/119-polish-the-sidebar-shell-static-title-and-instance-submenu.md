# [console]: polish the sidebar shell: static title and instance submenu

- **Plan id:** issue 119
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#119](https://github.com/dimasbaguspm/noxis/issues/119)
- **Pull request:** [#128](https://github.com/dimasbaguspm/noxis/pull/128)
- **Commit:** [f1d9bf4](https://github.com/dimasbaguspm/noxis/commit/f1d9bf452a8c5f8971e756602e2382beb0275e99) — `[ai:y][#119] feat(web): static site title and instances submenu in the sidebar (#128)`

## Plan

## What's problem?

The sidebar still carries the team-switcher dropdown (dashboard-01 pattern) and Instances is only a static link, while the dashboard already lists every registered instance.

## What's affected and how?

- The site title becomes static: brand tile + "noXis" + version, no dropdown.
- Instances becomes a collapsible submenu listing every registered instance (from the cached instances query), each linking to its detail page, with the dashboard itself as the first entry ("All instances").
- The submenu stays open while on an instance detail page.

## Definition of Done

- Clicking the site title does nothing (no menu).
- Instances has a submenu of instance names; choosing one opens its detail; the dashboard entry leads to the dashboard.
- Active instance highlights while its detail is open.

## Additional Notes

- Blocked by nothing.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
