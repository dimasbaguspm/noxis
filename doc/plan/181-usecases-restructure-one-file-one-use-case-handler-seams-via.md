# [backend] usecases restructure: one file = one use case, handler seams via interfaces

- **Plan id:** issue 181
- **Area:** Backend
- **Status:** completed 2026-10-10
- **Issue:** [#181](https://github.com/dimasbaguspm/noxis/issues/181)
- **Pull request:** [#183](https://github.com/dimasbaguspm/noxis/pull/183)
- **Commit:** [9be1766](https://github.com/dimasbaguspm/noxis/commit/9be17664abab4a91e6227d58f0d7d4f6a4fe8962) — `[ai:y][#181] refactor(backend): one file = one use case, handler seams via workflow interfaces (#183)`

## Plan

## What's problem?

`internal/usecases` is one flat directory around a god-struct: ~30 workflow methods on `*usecases.App`, and handlers depend on that concrete struct. The package name promises one file = one use case, but nothing delivers it.

## What's affected and how?

Structural restructure — behavior identical, swagger untouched (same methods, same routes). Files as designed:

- **`usecase.go`**: package doc + `type Usecase struct { *Deps; *Policy }` base every use case embeds (deps + authorize helpers + identity).
- **1 file = 1 use case**: `submit_change.go`, `approve_change.go`, `apply_change.go`, `reject_change.go`, `schedule_change.go`, `tick_scheduler.go`, `list_changes.go`, `get_change.go`; connection CRUD + `test_connection.go`, `sql_op.go` (query/alter/rollback/analyze), `list_tables.go`, `list_rows.go`; mongo 4; instances 3; grants 4. Each: `type X struct{ *Usecase }` + one `Serve(ctx, …entity.X) (…, error)`.
- **Infra stays package-level** (not use cases): `identity.go`, `policy.go`, `engine.go`, `deps.go` (ctx DI unchanged).
- **Handler seams in `internal/interfaces`**: `ChangesWorkflows`, `ConnectionsWorkflows`, `InstanceWorkflows`, `GrantWorkflows`, `MongoWorkflows`; `usecases.Deps` swaps `App *App` for these five. Facade composites in usecases satisfy them; `app.go` deleted.
- `app.go`→`usecase.go` rename lands here (app.go disappears entirely).

## Definition of Done

- Every workflow method lives in exactly one use-case file with one `Serve`.
- Handlers reference only the interface seams — no `*usecases.App` anywhere in `cmd/`.
- Compile-time satisfier asserts for the facades; fakes in handler tests satisfy the seams.
- UAT + unit + lint green; zero swagger/web delta.

## Additional Notes

- No DTOs: entity types are the in/out.
- Subdirs (`usecases/engine/...`) explicitly deferred until a file exceeds ~300 lines.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
