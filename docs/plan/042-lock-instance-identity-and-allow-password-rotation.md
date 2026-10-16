# [security]: lock instance identity and allow password rotation

- **Plan id:** issue 42
- **Area:** API & auth
- **Status:** completed 2026-10-09
- **Issue:** [#42](https://github.com/dimasbaguspm/noxis/issues/42)
- **Pull request:** [#46](https://github.com/dimasbaguspm/noxis/pull/46)
- **Commit:** [bce9348](https://github.com/dimasbaguspm/noxis/commit/bce9348e6c093e4ac7019cca3602ae5a2e2cd109) — `[ai:y][#42] feat: lock instance identity and rotate passwords (#46)`

## Plan

## What's problem?

Any connection field can be edited after registration, so an instance's identity (driver, host, database, user) can silently drift. But the password must still be changeable when credentials rotate.

## What's affected and how?

- Updating a connection rejects changes to driver, host, database or user with a clear error.
- A new admin-only password rotation action updates just the password.
- The instance detail page shows a notice that identity fields are fixed, and offers rotation.
- The registration wizard notes that identity fields cannot change later.

## Definition of Done

- Attempting to change a locked field is rejected and the stored instance is unchanged.
- Password rotation succeeds and the connection still authenticates.
- The detail page shows the locked notice and the rotation control (admins only).
- The wizard shows the warning before saving.

## Additional Notes

- Blocked by #40 (the detail page hosts the rotation control).

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
