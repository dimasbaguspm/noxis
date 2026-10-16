# [backend] restructure: usecases + modules layers, in-memory cache repo

- **Plan id:** issue 168
- **Area:** Backend
- **Status:** completed 2026-10-10
- **Issue:** [#168](https://github.com/dimasbaguspm/noxis/issues/168)
- **Pull request:** [#170](https://github.com/dimasbaguspm/noxis/pull/170)
- **Commit:** [c45beb4](https://github.com/dimasbaguspm/noxis/commit/c45beb48aced0c3318a3ee88f70b7c171548f436) — `[ai:y][#168] refactor(backend): usecases + modules layers, in-memory cache repo (#170)`

## Plan

## What's problem?

The C4 v2 target flow is `main → bootstrap → rest → handler → usecase → module → repositories → common/constants/interfaces/entity`. Today the layers exist under names that don't match: business workflows live in `internal/application` and engine adapters in `internal/services` (both mislabeled for the target), and there is no in-memory cache under `repositories`.

## What's affected and how?

Pure structure move — no behavior change:

- `internal/application/` → `internal/usecases/` (package rename; workflows, policy and identity content unchanged).
- `internal/services/{auth,memcache,mongodb,sql}` → `internal/modules/...` + new `internal/modules/modules.go` registry (`Default()` map) that moves module construction out of `cmd/noxis/server/server.go` (bootstrap imports only `internal/modules`).
- New `internal/repositories/in-memory-cache`: tested TTL KV store (lazy expiry). No consumer yet — sits in the Deps registry for future memoization. `ponytail:` comment names the upgrade point.
- Callers updated: `server.go` wiring, `middleware.go` (`WithPrincipal`), `api/` handlers, all tests, span-name expectations (`Application.*` → `Usecases.*`), swagger/docs regenerate (`application.*` → `usecases.*` only).

## Definition of Done

- `go build`/`vet` green, unit tests green, `golangci-lint` 0 issues.
- UAT passes (unchanged behavior).
- Swagger diff is package-name renames only; web client regenerated.
- All moves done with `git mv` (history preserved); no dead imports; no behavioral delta.

## Additional Notes

- `modules.go` is the registry file (constructors + `Default()`); implementations live in per-family files.
- `internal/modules/auth` stays a `Verifier` built at boot with config (not part of `Default()`).

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
