# [console]: add an Empty state for not-found pages

- **Plan id:** issue 63
- **Area:** Console (web)
- **Status:** completed 2026-10-09
- **Issue:** [#63](https://github.com/dimasbaguspm/noxis/issues/63)
- **Pull request:** [#70](https://github.com/dimasbaguspm/noxis/pull/70)
- **Commit:** [f406721](https://github.com/dimasbaguspm/noxis/commit/f406721bebfb9a3e5572e3e9eade0991149a7544) — `[ai:y][#63] feat: add a shared empty state (#70)`

## Plan

## What's problem?

Not-found pages (instance, change) render ad-hoc messages in different styles; there is no shared empty state.

## What's affected and how?

- A shadcn-style Empty component is added.
- Instance and change not-found surfaces render it centered in the page.

## Definition of Done

- Missing instances and changes show a centered empty state.
- The empty component is reusable for the list surfaces too.

## Additional Notes

- none

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
