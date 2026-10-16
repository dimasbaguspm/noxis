# [console]: make the field set the single form-field API

- **Plan id:** issue 99
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#99](https://github.com/dimasbaguspm/noxis/issues/99)
- **Pull request:** [#106](https://github.com/dimasbaguspm/noxis/pull/106)
- **Commit:** [341e002](https://github.com/dimasbaguspm/noxis/commit/341e00250ea239f5a862a34b75b9b22e0ef16db2) — `[ai:y][#99] refactor: make the field set the single form-field API (#106)`

## Plan

## What's problem?

Forms render fields three different ways: some use the shared Field set, some hand-roll label/input/error markup, so layouts and error styling drift across the app.

## What's affected and how?

- Every form field is rendered through the shared field API, in one order: label, control, description, error.
- A FieldLayout gives a consistent grid/spacing so rows and stacks look the same everywhere.
- Validation errors always render under the control through the shared FieldError (role=alert + destructive styling).
- Forms that migrated later (connection, submit-change, mongodb request) are converted to the shared API; the non-form UI labels stay as they are.

## Definition of Done

- No form hand-rolls a label+control+error trio outside the shared field API.
- Every form error (client and server-side) appears under its field through FieldError.
- Screenshots/demo: two different forms look consistent in spacing and error styling.

## Additional Notes

- Blocked by the sheet-structure ticket (uses its footer); otherwise start anytime.

- Blocked by #94.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
