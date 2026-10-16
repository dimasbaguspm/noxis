# [console][web] sidebar asChild menu items render as empty rows

- **Plan id:** issue 159
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#159](https://github.com/dimasbaguspm/noxis/issues/159)
- **Pull request:** [#160](https://github.com/dimasbaguspm/noxis/pull/160)
- **Commit:** [bba80c7](https://github.com/dimasbaguspm/noxis/commit/bba80c76017547e1204c3b83bee02dc7a07427ea) — `[ai:y][#159] fix(web): render asChild sidebar menu items as links (#160)`

## Plan

## What's problem?
Every asChild sidebar item (brand noXis, Instances, Profile, Settings) renders as an empty row: the tooltip-trigger span exists but has no child, so those items are invisible in the sidebar.

## What's affected and how?
SidebarMenuButton (web/src/components/ui/sidebar.tsx) always emits `{cond ? <span className={"sr-only"} /> : null}{children}`, so Slot receives an array of two children `[null, <Link>]` (or the sr-only span when collapsed). Slot only accepts a single valid element and returns null for arrays, so every asChild menu item disappears. Non-asChild (button) items are unaffected.

## Definition of Done
- The sidebar brand, Instances, Profile and Settings render as links.
- The sr-only label still exists on icon-only (collapsed) buttons.
- Verified with a renderToStaticMarkup repro: asChild -> <a>, button collapsed -> sr-only intact.

## Additional Notes
Submenu links (SidebarMenuSubButton) and group labels were unaffected because they pass a single child to Slot.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
