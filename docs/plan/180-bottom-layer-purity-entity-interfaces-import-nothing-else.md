# [backend] bottom layer purity: entity + interfaces import nothing else

- **Plan id:** issue 180
- **Area:** Backend
- **Status:** completed 2026-10-10
- **Issue:** [#180](https://github.com/dimasbaguspm/noxis/issues/180)
- **Pull request:** [#182](https://github.com/dimasbaguspm/noxis/pull/182)
- **Commit:** [069ed74](https://github.com/dimasbaguspm/noxis/commit/069ed74ed3f92ca35ae548b8dd03d955b09f3b33) — `[ai:y][#180] refactor(backend): bottom-layer purity — modules own satisfier asserts, entity imports nothing internal (#182)`

## Plan

## What's problem?

`internal/interfaces/` and `internal/entity/` are supposed to be the bottom layer — yet interfaces imports the modules (only for compile-time satisfier asserts) and entity imports `constants` (driver-name literals in a params rule). Any internal import from the bottom layer is a latent circular-import risk.

## What's affected and how?

Pure cleanup — no behavior change, no swagger delta:

- `internal/interfaces`: drop the 3 satisfier asserts (`var _ SQLModule = (*sqlsvc.DB)(nil)`, mongo, memcache) and their module imports; move each assert into its module file (`var _ interfaces.SQLModule = (*DB)(nil)`), where module→interfaces is the legal direction.
- `internal/entity`: remove the `constants` import from connection.go (inline the two driver literals used in the params rule — they are domain names).
- Entity then imports only stdlib + validator; interfaces imports only entity; the bottom stays acyclic.
- `grep`-verified at the end: zero `noxis/internal/...` imports in entity other than none, and none in interfaces other than entity.

## Definition of Done

- `go list -deps` check: entity and interfaces pull in no package under `noxis/internal/modules` (or any sibling).
- Build/vet/tests/lint/UAT green; swagger + web untouched.

## Additional Notes

- The use-case-per-file restructure (separate ticket) builds on this clean bottom.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
