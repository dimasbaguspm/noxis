# [api]: verify DDL and DML in a scratch schema

- **Plan id:** issue 9
- **Area:** API & auth
- **Status:** completed 2026-10-08
- **Issue:** [#9](https://github.com/dimasbaguspm/noxis/issues/9)
- **Pull request:** [#15](https://github.com/dimasbaguspm/noxis/pull/15)
- **Commit:** [611388a](https://github.com/dimasbaguspm/noxis/commit/611388ab9b3411dc296195bffb0d0d849501cdbb) — `[ai:y][#9] feat: verify DDL and DML in a scratch schema (#15)`

## Plan

## What's problem?

Verification is only as good as the environment it runs in. Today a change and its rollback are dry-run inside one transaction that always rolls back. MySQL/MariaDB DDL is marked `unverified` because it commits implicitly, and the dry run never proves the DDL really creates the structure it claims.

## What's affected and how?

- Verification builds a throwaway scratch schema on the target server and applies the change there (structure only — no table data is copied), then drops it. It follows what the change creates: the objects the DDL references are mirrored in the scratch schema.
- Works for both DDL and DML and for postgres, mysql and mariadb, so MySQL DDL stops being `unverified` when the scratch run succeeds.
- The scratch run logs the objects/statements it created (and the driver) as the change's check detail; the change records the scratch schema it was verified against.
- Cleanup is guaranteed: the scratch schema is dropped on success and on failure.

## Definition of Done

- [ ] A DDL change on postgres and on mysql/mariadb verifies by applying it in a scratch schema, not the live schema.
- [ ] A DML change verifies in the scratch schema too.
- [ ] The scratch schema holds structure only; no table data is copied.
- [ ] The change's checks record the scratch schema, the driver and the objects the verification created.
- [ ] The scratch schema is dropped after success and after failure (no orphans).
- [ ] An invalid change fails with the database's error on every supported driver.
- [ ] Existing unit + UAT suites pass.

## Additional Notes

- Structure-only means DML is proven against empty structures: syntax, columns and constraints, not data semantics. Ceiling accepted for now.
- Batch verification reuses this verifier.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
