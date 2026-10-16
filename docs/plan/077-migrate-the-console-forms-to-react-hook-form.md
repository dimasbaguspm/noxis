# [console]: migrate the console forms to react-hook-form

- **Plan id:** issue 77
- **Area:** Console (web)
- **Status:** completed 2026-10-09
- **Issue:** [#77](https://github.com/dimasbaguspm/noxis/issues/77)
- **Pull request:** [#78](https://github.com/dimasbaguspm/noxis/pull/78)
- **Commit:** [259a6b4](https://github.com/dimasbaguspm/noxis/commit/259a6b47beb987438026aa9024535b0a79b0277f) — `[ai:y][#77] refactor: migrate the console forms to react-hook-form (#78)`

## Plan

## What's problem?

Several console forms manage their fields by hand with `useState`, each repeating the same juggling. The create-change form now uses react-hook-form; the rest should follow so there is one form pattern.

## What's affected and how?

- Migrate the remaining multi-field forms: the new-connection dialog, the change-submit (inverse) dialog, and the batch upload dialog.
- Single-field controls (password rotate, search inputs) stay as they are.
- Per-field validation can move into the form library as part of the migration.

## Definition of Done

- The listed forms use react-hook-form; no manual `useState` field juggling remains in them.
- Typecheck, lint and build pass.

## Additional Notes

- None.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
