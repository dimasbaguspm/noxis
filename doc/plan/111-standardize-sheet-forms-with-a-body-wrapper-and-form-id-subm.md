# [console]: standardize sheet forms with a body wrapper and form-id submit

- **Plan id:** issue 111
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#111](https://github.com/dimasbaguspm/noxis/issues/111)
- **Pull request:** [#115](https://github.com/dimasbaguspm/noxis/pull/115)
- **Commit:** [72ecc8e](https://github.com/dimasbaguspm/noxis/commit/72ecc8ed036cffcc7efeea1f7c4b371c0d32363c) — `[ai:y][#111] refactor: wrap sheet forms in the body and submit by form id (#115)`

## Plan

## What's problem?

Sheet bodies are a single flex-1 scroll area and the form (plus footer) sits directly inside it, so content stretches awkwardly and footer layout fights the body.

## What's affected and how?

- Every sheet form becomes: SheetHeader → SheetBody wrapping a single <form id="..."> (the form owns its spacing) → SheetFooter.
- The footer's primary action submits via the plain HTML association type="submit" form="<id>", so the button lives outside the form element.
- Applied to: create change, submit change, connection dialog (every step), schedule change.

## Definition of Done

- No sheet form wraps SheetBody or SheetFooter inside <form>; the footer button links through form="<id>".
- The footer stays pinned while the body scrolls.
- No visual regression.

## Additional Notes

- Lands before the zod ticket so each form is touched exactly once.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
