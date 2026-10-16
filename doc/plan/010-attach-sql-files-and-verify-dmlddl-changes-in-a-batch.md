# [web]: attach SQL files and verify DML/DDL changes in a batch

- **Plan id:** issue 10
- **Area:** Console (web)
- **Status:** completed 2026-10-08
- **Issue:** [#10](https://github.com/dimasbaguspm/noxis/issues/10)
- **Pull request:** [#18](https://github.com/dimasbaguspm/noxis/pull/18)
- **Commit:** [e8d5d47](https://github.com/dimasbaguspm/noxis/commit/e8d5d47f56b5ce1c6f1c1124317f1e8f3a87d4e3) — `[ai:y][#10] feat: attach .sql files and verify DML/DDL in a batch (#18)`

## Plan

## What's problem?

A change is written by hand in the editor. There is no way to attach a `.sql` file, and no way to submit and verify several statements as one batch.

## What's affected and how?

- A DML/DDL change can attach `.sql` files; one file is one statement. Uploading fills the field (original or rollback) from the file's contents.
- Several files uploaded together are submitted as a batch: each file becomes its own change, sharing a batch id, and the batch is verified with the scratch-schema verifier.
- A file carrying more than one statement is rejected with a clear error, so "one file = one statement" stays honest.
- The batch result shows pass/fail per file.

## Definition of Done

- [ ] A `.sql` file can be attached to a DML/DDL change and fills the field from its contents.
- [ ] A file with more than one statement is rejected with a clear message.
- [ ] Uploading several files submits several changes sharing one batch id, each verified.
- [ ] The batch report shows pass/fail per file.
- [ ] `pnpm typecheck`, `pnpm lint`, `pnpm build` pass; Go suites pass.

## Additional Notes

- Blocked by #9 (the scratch-schema verifier).
- Assumption: a batch is N files → N changes sharing a batch id; say so if you'd rather concatenate into one change.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
