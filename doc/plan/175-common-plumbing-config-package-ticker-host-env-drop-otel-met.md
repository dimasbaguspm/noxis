# [backend] common plumbing: config package, ticker host env, drop otel metrics, shared limits

- **Plan id:** issue 175
- **Area:** Backend
- **Status:** completed 2026-10-10
- **Issue:** [#175](https://github.com/dimasbaguspm/noxis/issues/175)
- **Pull request:** [#177](https://github.com/dimasbaguspm/noxis/pull/177)
- **Commit:** [dd7a640](https://github.com/dimasbaguspm/noxis/commit/dd7a640d793888f4ffea9e3f3e17d9e1000a09b2) — `[ai:y][#175] refactor(backend): config in common, ticker host env, drop otel metrics, shared limits (#177)`

## Plan

## What's problem?

Backend plumbing is scattered: config lives in `internal/entity` (env config is not a domain entity), secret/open-mode warnings sit inline in `server.Run`, the ticker hardcodes a full URL (host should be env), logging wires a whole OTLP metric provider that nothing ever uses, middleware tests are one monolith file, and paging/scan limits repeat as literals.

## What's affected and how?

Pure plumbing — no API surface change (no swagger/web delta):

- **`internal/common/config`** (new): move `entity/config.go` + `config_test.go` there (env consts, `Config`, `LoadConfig`, `Require`, `String`). Add `LogState(log, cfg)` centralizing the state warnings (plaintext secret, open mode, OIDC on) — called once in `bootstrap`; `server.Run` drops its inline `if cfg.SecretKey == ""` blocks. Update every `entity.Config` importer (server, ticker, bootstrap, logging, modules/auth).
- **Ticker host from env**: new `NOXIS_TICK_HOST` (required for the ticker command; compose sets `http://noxis:9000`); endpoint `/internal/callback/scheduler` hardcoded in ticker; full-URL `schedulerURL` var + its test override gone.
- **Drop otel metrics**: remove `otlpmetricgrpc` + `sdk/metric` wiring from `common/logging` (no code emits metrics); `New`/`Shutdown` keep tracer + logger + ctxHandler.
- **`internal/constants/limits.go`**: `MaxPageSize`=100, `DefaultPageSize`=20, `ScanLimit`=500, `MaxScanLimit`=5000; used by `handlers/pagination.go`, changes/instances, and the memcache module. Single-use timeouts stay module-local.
- **Middleware tests per file**: delete `middleware_test.go` → `auth_test.go` (WithAuth: open mode, guard, bearer, principal attach), `cors_test.go` (preflight 204 + headers), `logging_test.go` (RequestLog status/attr capture). One middleware = one file = one test file.

## Definition of Done

- `config` importable from `internal/common/...` without cycles.
- `NOXIS_TICK_HOST` drives the ticker; endpoint path hardcoded.
- No metric imports in `common/logging`; UAT + unit tests + lint green; swagger unchanged.

## Additional Notes

- `LogState` lives in config but is fed the slog logger from bootstrap (logging init order unchanged).
- `AuthInfo` move to config is deliberately NOT in this ticket (kept in the entity reshuffle so swagger regenerates once).

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
