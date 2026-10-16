# [console]: rebuild the connection wizard with radio-backed selects

- **Plan id:** issue 123
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#123](https://github.com/dimasbaguspm/noxis/issues/123)
- **Pull request:** [#132](https://github.com/dimasbaguspm/noxis/pull/132)
- **Commit:** [460b420](https://github.com/dimasbaguspm/noxis/commit/460b420fca9a06e6e221f3862927a50743014359) — `[ai:y][#123] feat(web): radio-backed selects and a simplified connection wizard (#132)`

## Plan

## What's problem?

The connection wizard picks its kind with three styled buttons, then hand-groups fields with layout helpers — neither feels like a form control.

## What's affected and how?

- Two new primitives join the ui folder: SingleSelect (radio-backed) and MultiSelect (checkbox-backed), both keyboard-accessible lists.
- Kind selection becomes a SingleSelect (one of: SQL database / memory cache / document database).
- The field groupings are removed: the form element itself defines the layout (one stacked column of fields).
- The muted helper line in step 2 is removed.

## Definition of Done

- The kind step renders a radio-backed single-select input; choosing one proceeds to step 2.
- Step 2 fields stack in the form's own grid; no layout helpers wrap them; the muted hint is gone.

## Additional Notes

- Blocked by nothing. The MultiSelect is also the future home of multi-status filtering.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
