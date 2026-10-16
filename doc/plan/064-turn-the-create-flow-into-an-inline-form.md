# [changes]: turn the create flow into an inline form

- **Plan id:** issue 64
- **Area:** Changes & query
- **Status:** completed 2026-10-09
- **Issue:** [#64](https://github.com/dimasbaguspm/noxis/issues/64)
- **Pull request:** [#73](https://github.com/dimasbaguspm/noxis/pull/73)
- **Commit:** [532fd7c](https://github.com/dimasbaguspm/noxis/commit/532fd7c92580c394f59ca346bda59a5057d72066) — `[ai:y][#64] feat: create changes inline and prefill the inverse (#73)`

## Plan

## What's problem?

Raising a change still opens a wizard dialog with steps, but the form is short enough to sit directly on the page; and raising the inverse of a reviewed change does not prefill what a human would naturally carry over.

## What's affected and how?

- The wizard dialog is replaced by a plain inline form on the DML/DDL queue page: instance, notes, original query, rollback query, then submit.
- The "raise inverse change" action prefills the new original query with the old rollback and the new rollback with the old query, and prefills the notes with a `[Rollback]` prefix.

## Definition of Done

- Creating a change happens inline on the queue page; no modal is involved.
- Inverse change prefills work and the notes start with `[Rollback] `.

## Additional Notes

- Blocked by the current open PR batch (#57/#58) which hosts the wizard and the inverse action on the same pages.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
