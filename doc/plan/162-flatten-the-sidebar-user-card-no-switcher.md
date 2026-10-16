# [console][web] flatten the sidebar user card (no switcher)

- **Plan id:** issue 162
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#162](https://github.com/dimasbaguspm/noxis/issues/162)
- **Pull request:** [#164](https://github.com/dimasbaguspm/noxis/pull/164)
- **Commit:** [dcc7f04](https://github.com/dimasbaguspm/noxis/commit/dcc7f04132f916a2f2ace08ec5170a32d51605d7) — `[ai:y][#162] refactor(web): flatten the sidebar user card (#164)`

## Plan

## What's problem?
The sidebar footer user card is a dropdown switcher (chevron + Profile/Sign out menu) and shows its full content as a hover tooltip when collapsed.

## What's affected and how?
web/src/components/layout/nav-user.tsx. The spec wants just the identity: avatar, name, email.

## Definition of Done
- Avatar + name + email only.
- No chevron, no dropdown menu, no tooltip.
- typecheck/lint/build pass.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
