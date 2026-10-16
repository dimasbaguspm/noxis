# [ai:y] feat(web): port the sidebar to the dashboard-01 structure

- **Plan id:** pr 118
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** —
- **Pull request:** [#118](https://github.com/dimasbaguspm/noxis/pull/118)
- **Commit:** [a2b1881](https://github.com/dimasbaguspm/noxis/commit/a2b18819533cd510e8520973e4d4fcfd252bad18) — `[ai:y] feat(web): port the sidebar to the dashboard-01 structure (#118)`

## Plan

## What problem that I am facing?

The sidebar had the shell right but none of the dashboard-01 richness: no site title switcher, no labelled group, and the family menus sat flat with no icons or chevrons.

## How I implement it?

- SidebarHeader is now the dashboard-01 site title: the brand tile + "noXis" + version open a dropdown that jumps into any family.
- SidebarContent holds one labelled group ("Console") whose items carry icons: Instances as a direct link, and each engine family as an icon + chevron that expands its SidebarMenuSub.
- Families open by default while one of their pages is active.
- The rail stays; sub items highlight on their own route.

## Why does it match the reference now?

Menu and submenu behave like dashboard-01 (icons, group label, collapsible chevrons, title switcher), while still deriving everything from lib/families.

## Risks

The collapsible is a tiny local accordion (open-state per family), not a radix Collapsible — no animation, same behavior.

This is a follow-up polish PR to the merged console sidebar (no ticket).

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
