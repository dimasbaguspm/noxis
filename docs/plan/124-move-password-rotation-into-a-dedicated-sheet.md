# [console]: move password rotation into a dedicated sheet

- **Plan id:** issue 124
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#124](https://github.com/dimasbaguspm/noxis/issues/124)
- **Pull request:** [#131](https://github.com/dimasbaguspm/noxis/pull/131)
- **Commit:** [1e37657](https://github.com/dimasbaguspm/noxis/commit/1e37657d65d2964c2b9ce0d0b62f77a17b2a318c) — `[ai:y][#124] feat(web): move password rotation into a dedicated sheet (#131)`

## Plan

## What's problem?

Password rotation sits inline in the instance detail card, mixing a form field into a read-only sheet of facts.

## What's affected and how?

- The card gets a "Rotate password" CTA that opens a dedicated sheet: one password field, zod-validated, submit in the sheet footer (the #111 form-id pattern).
- Success clears the field, shows a toast and invalidates the instances cache as today.

## Definition of Done

- Rotation happens only through its own sheet; the card no longer contains the inline field.

## Additional Notes

- Blocked by the sheet/field patterns already on master.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
