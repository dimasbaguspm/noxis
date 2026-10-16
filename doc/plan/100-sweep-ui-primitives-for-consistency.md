# [console]: sweep UI primitives for consistency

- **Plan id:** issue 100
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#100](https://github.com/dimasbaguspm/noxis/issues/100)
- **Pull request:** [#107](https://github.com/dimasbaguspm/noxis/pull/107)
- **Commit:** [96a1580](https://github.com/dimasbaguspm/noxis/commit/96a1580917161723d0bc19ef2d36055ccc28b8ef) — `[ai:y][#100] chore: sweep UI primitives into the shared components (#107)`

## Plan

## What's problem?

A handful of raw HTML controls (button, input, select, textarea) are used directly outside the shared ui components, so controls can drift from the design system.

## What's affected and how?

- All direct native controls outside the ui folder move to the shared components (Button, Input, Textarea, Select, …).
- Dead UI variants left over from earlier rounds are removed where the sweep proves them unused.

## Definition of Done

- No native button/input/select/textarea is rendered directly outside the ui folder (editor internals may keep their textarea where a component would not help).
- The build stays green (typecheck, lint, tests).

## Additional Notes

- Blocked by the earlier sweep tickets, so a final mechanical pass lands last on a stable tree.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
