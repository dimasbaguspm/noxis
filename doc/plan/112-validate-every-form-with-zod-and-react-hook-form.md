# [console]: validate every form with zod and react-hook-form

- **Plan id:** issue 112
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#112](https://github.com/dimasbaguspm/noxis/issues/112)
- **Pull request:** [#116](https://github.com/dimasbaguspm/noxis/pull/116)
- **Commit:** [a9ec371](https://github.com/dimasbaguspm/noxis/commit/a9ec3711168ccb75bbd8a2d0c21783fee358071e) — `[ai:y][#112] feat: validate every form with zod and react-hook-form (#116)`

## Plan

## What's problem?

Client-side validation is ad-hoc: required rules on register(), manual guards and disabled buttons — so invalid submits either fire or only shout from a toast.

## What's affected and how?

- zod + @hookform/resolvers join the project; every form gets a zod schema wired through useForm's resolver.
- Forms to convert: create change, submit change, connection dialog, mongodb request wizard, schedule at, rotate password, flush confirmation.
- An invalid submit is blocked by validation (no onSubmit, no request, no toast); every failure renders under its field through FieldError; valid submits keep surfacing server-side checks the existing way.

## Definition of Done

- Clicking Submit with invalid values shows inline FieldError messages and does not call onSubmit.
- No client-side validation relies on toasts, disabled buttons, or register()-string rules anymore.
- Every control above is schema-validated, single-input ones included.

## Additional Notes

- Blocked by the sheet-forms ticket (same files; structure first).

- Blocked by #111.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
