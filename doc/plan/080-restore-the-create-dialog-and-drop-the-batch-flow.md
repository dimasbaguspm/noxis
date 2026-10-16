# [changes]: restore the create dialog and drop the batch flow

- **Plan id:** issue 80
- **Area:** Changes & query
- **Status:** completed 2026-10-09
- **Issue:** [#80](https://github.com/dimasbaguspm/noxis/issues/80)
- **Pull request:** [#92](https://github.com/dimasbaguspm/noxis/pull/92)
- **Commit:** [4a6e010](https://github.com/dimasbaguspm/noxis/commit/4a6e010b5edf72c366507c5331b460aae4a7647a) — `[ai:y][#80] refactor: drop batch upload and rebuild the create sheet (#92)`

## Plan

## What's problem?

The DML/DDL create form sat inline on the queue page and the create experience was cramped; a mid-flight plan to fold a `.sql` batch into one sheet added more surface than it was worth.

## What's affected and how? (updated)

- The DML/DDL "Create change" opens a right-side Sheet with exactly: instance, notes, original query (loadable from an attached .sql file), rollback query (same), and one Submit.
- Validation errors render below the field: react-hook-form required errors and the server's failed checks.
- A reusable Field component set (Field, FieldLabel, FieldDescription, FieldError, FieldSet) is added so every form renders errors the same way.
- The batch flow is dropped entirely: the batch endpoint, controller, storage wiring, generated types, UAT scenarios and the frontend batch UI are removed.

## Definition of Done

- One "Create change" Sheet on DML and DDL; no inline form remains.
- Errors appear under the field they belong to.
- No batch endpoint, types or UAT scenarios remain.

## Additional Notes

- Blocked by #79 (the Sheet component lands there).

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
