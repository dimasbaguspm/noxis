# [api]: retire SSE and instance metrics endpoints

- **Plan id:** issue 19
- **Area:** API & auth
- **Status:** completed 2026-10-08
- **Issue:** [#19](https://github.com/dimasbaguspm/noxis/issues/19)
- **Pull request:** [#27](https://github.com/dimasbaguspm/noxis/pull/27)
- **Commit:** [4be2014](https://github.com/dimasbaguspm/noxis/commit/4be2014bc36dc0cb2e7075dfc9b402feeab5e52c) — `[ai:y][#19] refactor: retire SSE and instance metrics endpoints (#27)`

## Plan

## What's problem?

The API streams live metrics over SSE (`GET /v1/{engine}/connections/{name}/stats/stream` and `GET /v1/connections/status/stream`) plus a one-shot `GET .../stats`. The metrics Analytics console pages are being removed, so these endpoints have no consumer, and SSE is a poor fit for the incoming bearer-only auth (native `EventSource` cannot send an `Authorization` header).

## What's affected and how?

- Delete `cmd/noxis/server/controller/stats.go` in full (`GetStats`, `StreamStats`, `streamSnapshots`, `statsInterval`).
- Delete `cmd/noxis/server/controller/status_stream.go` in full (`StreamConnectionStatus`, `connectionStates`, `ConnectionState`).
- Remove the three routes from `routes.go`.
- Remove `Stats` from `interfaces.SQLModule`, `MemcacheModule`, `MongoModule` and their type aliases.
- Delete the service implementations: `services/sql/stats.go` (whole), `sql.DB.Stats`, `memcache.DB.Stats` + `parseInfo` + `Stats` type, `mongodb.DB.Stats` + `Stats` type. The only callers live in `controller/stats.go`.
- Remove the swagger annotations and regenerate `doc/swagger.yaml|json`.

## Definition of Done

- [ ] No SSE route remains in `routes.go`.
- [ ] No `Stats` method on any module interface or service.
- [ ] `go build ./...`, `go vet`, `make lint` pass.
- [ ] `make uat` passes unchanged.
- [ ] swagger regenerates cleanly with no `stats`/`stream` paths.

## Additional Notes

- Split from the OIDC+RBAC effort: shave the surface before authz lands.
- `POST /analytics` (planner `ANALYZE`) is unrelated and stays — see the rename ticket.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
