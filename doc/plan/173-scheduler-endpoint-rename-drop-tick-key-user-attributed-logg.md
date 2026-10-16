# [backend] scheduler endpoint rename, drop tick key, user-attributed logging + full-layer spans

- **Plan id:** issue 173
- **Area:** Backend
- **Status:** completed 2026-10-10
- **Issue:** [#173](https://github.com/dimasbaguspm/noxis/issues/173)
- **Pull request:** [#174](https://github.com/dimasbaguspm/noxis/pull/174)
- **Commit:** [7b14325](https://github.com/dimasbaguspm/noxis/commit/7b143259822b800f7091b2e8f4ba04aa327c23df) — `[ai:y][#173] feat(backend): keyless scheduler endpoint, user-attributed logs, spans at every layer (#174)`

## Plan

## What's problem?

Three backend concerns: (1) `POST /internal/tick` carries an api-key gate (`NOXIS_TICK_KEY`) that no one needs — the endpoint should behave like any other internal route, unguarded; (2) the endpoint name is short and not "callback" flavoured; (3) logs and traces are not attributable to a user, and spans stop at the usecase layer (no request span, no repository spans).

## What's affected and how?

- **Drop TickKey entirely**: remove `EnvTickKey`/field/parse/serialize from `entity.Config`, `usecases.Deps.TickKey`, `newDeps`, the key guard + `X-API-Key` swagger param in `handlers.PostTick`, UAT harness/key header + compose stack env. Endpoint stays reachable (open mode + auth mode: it was never under `WithAuth`).
- **Rename endpoint**: `/internal/tick` → `/internal/callback/scheduler`; handler `PostTick` → `PostScheduler` (+ swagger `@Router`, `@Summary`), ticker package comment + bootstrap CLI description, UAT tick step URL.
- **Principal-attributed logging**: request log + handler/usecase span logs attach `user_id`, `groups`, `admin` whenever a principal is on the ctx (open mode: omitted). New `usecases.LogAttrs(ctx)` helper as the single source.
- **Request span**: `middlewares.RequestLog` wraps the whole request in a `logging.StartSpan` (convention `Noxis.Middlewares.*`).
- **Repository spans**: every `internal/repositories/sqlite` `Store` method opens `logging.StartSpan(ctx, CallerName(1))` and closes with its error (named-return `err` pattern).
- **Rule doc**: `usecases/identity.go` comment pins the principal contract — middleware sets actor in ctx; usecases read it for authorization; modules never read it.

## Definition of Done

- `NOXIS_TICK_KEY` gone from code, tests, UAT compose, docs, swagger.
- Endpoint works keyless (`POST /internal/callback/scheduler`); UAT green.
- Log lines show `user_id`/`groups`/`admin` when authenticated.
- Span convention test covers middlewares + repositories + usecases; all layers span: middleware → handlers → usecases → repositories.
- `go build`/`vet`/tests/lint green; web unaffected (route is internal-only; no generated-client delta beyond swagger regen).

## Additional Notes

- `NOXIS_TICK_URL` (the standalone ticker's target) is not part of this change but its comment references the old path — update the comment.
- Repo spans use `logging.StartSpan(ctx, logging.CallerName(1))`, identical signature everywhere per point 5.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
