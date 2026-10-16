# [ai:y] chore: drop unused tabs and textarea components

- **Plan id:** pr 93
- **Area:** Other
- **Status:** completed 2026-10-10
- **Issue:** —
- **Pull request:** [#93](https://github.com/dimasbaguspm/noxis/pull/93)
- **Commit:** [fc51217](https://github.com/dimasbaguspm/noxis/commit/fc51217ffd04afa43ae69f7c7bf0ba7a87aa7276) — `[ai:y] chore: drop unused tabs and textarea components (#93)`

## Plan

## What problem that I am facing?

Two UI components landed with the shadcn base set and never found a caller.

## How I implement it?

- Deleted `web/src/components/ui/tabs.tsx` and `web/src/components/ui/textarea.tsx` (zero imports anywhere).

## Why should I?

One less entry per dead component in every future grep, no false "ui exists, use it" temptation.

## Risks

None; typecheck, lint and build pass without them.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
