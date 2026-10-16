# [backend] restructure: handlers + middlewares packages, deps injected once via request ctx

- **Plan id:** issue 169
- **Area:** Backend
- **Status:** completed 2026-10-10
- **Issue:** [#169](https://github.com/dimasbaguspm/noxis/issues/169)
- **Pull request:** [#171](https://github.com/dimasbaguspm/noxis/pull/171)
- **Commit:** [d7d4b9b](https://github.com/dimasbaguspm/noxis/commit/d7d4b9b562f40ce2892d49f8c4c922a63907ac0c) — `[ai:y][#169] refactor(backend): plain handlers + middlewares, deps injected once via request ctx (#171)`

## Plan

## What's problem?

The transport layer mixes concerns: `cmd/noxis/server/api` is one package holding handlers, response glue and engine lookup behind a `Server` struct with method receivers; middleware is a single `middleware.go`; dependencies ride the Server struct, not the request. Target: plain `func(w, r)` handlers (1 file per entity), one middleware per file, deps registered **once after boot** and consumed from the passed ctx.

## What's affected and how?

Pure structure move — no behavior change:

- `cmd/noxis/server/api/` → `cmd/noxis/server/handlers/` (package `handlers`): handlers become plain functions; `Server` struct and method receivers removed.
- New `cmd/noxis/server/middlewares/`: `chain.go` (`withMiddleware`), `cors.go`, `logging.go`, `auth.go` (`withAuth`/`bearerToken`/`writeUnauthorized`) — one middleware per file.
- Deps registered once: `usecases.Deps{App, Log, Cache, Auth, TickKey}` seeded into the root request ctx at `server.Run` (via `srv.BaseContext` + `usecases.WithDeps`); handlers consume via `usecases.From(r)`. Never rebuilt per request.
- Engine lookup (`api/engine.go`) moves into usecases as `App` methods; `pagination`/`writeJSON`/`writeErr`/`decode` stay transport (handlers package).
- `server.go` routes rewritten to plain funcs; `Makefile`/`Dockerfile`/`.gitignore` static path `api/static` → `handlers/static`.
- Swagger regenerates (`api.*` → `handlers.*` renames only); web client + `lib/api.ts` identity alias updated.

## Definition of Done

- `go build`/`vet` green, unit tests green, `golangci-lint` 0 issues.
- UAT passes.
- Deps seeded once in `BaseContext` — no per-request rebuild anywhere.
- Swagger diff = package renames only.
- `internal/application` no longer exists (moved to `usecases` in #168).

## Additional Notes

- `WithDeps`/`From` live in `usecases` (importable by both `handlers` and `server` wiring without an import cycle).
- Web untouched except the regenerated client + one alias line.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
