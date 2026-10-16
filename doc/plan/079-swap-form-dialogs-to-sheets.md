# [console]: swap form dialogs to Sheets

- **Plan id:** issue 79
- **Area:** Console (web)
- **Status:** completed 2026-10-09
- **Issue:** [#79](https://github.com/dimasbaguspm/noxis/issues/79)
- **Pull request:** [#83](https://github.com/dimasbaguspm/noxis/pull/83)
- **Commit:** [e715dfa](https://github.com/dimasbaguspm/noxis/commit/e715dfac7c3854fcf0e1fdd4b1befc78e8fd9ee3) — `[ai:y][#79] refactor: swap form dialogs to Sheets (#83)`

## Plan

## What's problem?

The console's form dialogs are cramped centered modals; a right-side sheet gives the connection, inverse-change and schedule forms room to breathe.

## What's affected and how?

- A shadcn-style Sheet component is added (right-side panel on the same base-ui dialog primitive).
- The new-connection dialog, the change-submit (inverse) dialog and the schedule picker move to Sheets.
- The inverse dialog keeps seeding its notes with a `[Rollback] ` prefix on every open; the migration verifies that.

## Definition of Done

- The three forms open in right-side Sheets.
- Opening the inverse dialog shows notes prefixed with `[Rollback] `.
- Typecheck, lint and build pass.

## Additional Notes

- None.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
