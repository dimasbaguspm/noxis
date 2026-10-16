# [console]: restructure the sidebar on the shadcn sidebar set

- **Plan id:** issue 109
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#109](https://github.com/dimasbaguspm/noxis/issues/109)
- **Pull request:** [#113](https://github.com/dimasbaguspm/noxis/pull/113)
- **Commit:** [d02e4a0](https://github.com/dimasbaguspm/noxis/commit/d02e4a07c487a6b17c433b6782a39724fa12bd8a) — `[ai:y][#109] feat: restructure the sidebar on the shadcn sidebar set (#113)`

## Plan

## What's problem?

The sidebar is a hand-rolled aside with custom group toggles, so its markup does not match the shadcn sidebar pattern used across the design system.

## What's affected and how?

- A sidebar component set lands (Sidebar, SidebarProvider, SidebarHeader, SidebarContent, SidebarGroup, SidebarMenu, SidebarMenuItem, SidebarMenuButton, SidebarMenuSub, SidebarMenuSubItem, SidebarMenuSubButton, SidebarRail) following the shared shadcn sidebar API, with base-ui underneath and a thin Slot providing asChild behavior.
- The shell renders, in reference structure: SidebarHeader with the brand block (icon, product name, version), SidebarContent with one SidebarGroup whose nav items expand into SidebarMenuSub lists, and a SidebarRail.
- Active-link highlighting carries over; the Cmd+K page search stays reachable.

## Definition of Done

- The sidebar renders brand + groups + nested items + rail in the reference structure, exactly.
- Nested menus collapse/expand; the active item is highlighted; mobile collapses into the overlay sheet.
- No custom aside/toggle logic remains in the shell.
- Cmd+K search still opens.

## Additional Notes

- Reference structure must be followed without deviation (component names/API identical); asChild is implemented as a tiny Slot over base-ui render.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
