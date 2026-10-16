# AGENTS.md — noxis harness brief

Read this first in every session. It is the map; the four topic docs under
`doc/agent/` are the detail and the per-initiative history is under
`doc/plan/`. `README.md` is the authoritative current feature spec.

## What this repo is

`noxis` = one Go binary + embedded React console. Point it at databases you
already run (Postgres, MySQL/MariaDB, Redis, Valkey, MongoDB), browse them,
run reads directly, and ship writes through a reviewed change workflow. No
agents, no Kafka, no docker socket. State lives in one SQLite file.

## Next action (verify the tree before touching it)

```bash
make test        # go test ./...            (~30s)
make lint        # golangci-lint v2         (~60s)
make build       # bin/noxis
make web         # build console into embed dir (needs pnpm)
make uat         # godog over real compose stack (needs docker, tags ~@slow)
```

CI (`.github/workflows/ci.yml`) runs exactly: lint → docker build → unit →
`make uat` for Go; lint + typecheck + build for `web/`.

## Repo map (outer layer → inner layer)

| Path | Role |
| --- | --- |
| `cmd/noxis/` | process entry: CLI, server wiring, handlers, middlewares, ticker |
| `cmd/noxis/server/deps/` | runtime dependency container + DI (the only wiring) |
| `cmd/noxis/server/handlers/` | thin HTTP handlers + swag annotations + response glue |
| `cmd/noxis/server/middlewares/` | CORS → auth → request-log/span |
| `internal/usecases/` | one file = one use case; business flows only |
| `internal/modules/` | engine adapters: `sql`, `memcache`, `mongodb`, `auth` + `Registry` |
| `internal/repositories/` | persistence: `sqlite` (the only state) |
| `internal/interfaces/` | seams (`ConnectionRepository`, `Module`, `SQLModule`, …) |
| `internal/entity/` | shared data model (store + wire + REST) |
| `internal/common/` | config, logging/span, idgen, pagination, secret |
| `internal/constants/` | universal values (drivers, families, statuses, limits) |
| `test/uat/` | Gherkin acceptance suite (`make uat`) |
| `web/` | React + TS + shadcn/ui console (Vite, pnpm) |
| `doc/agent/` | agent-facing docs (this harness) |
| `doc/plan/` | one snapshot per `ai:y` plan, with an index |
| `doc/swagger.*` | generated OpenAPI (Postman artifact) |

Dependency direction is one-way: `handlers → usecases → modules → repositories → entity`.
`entity` imports nothing internal. Modules/repositories assert their seams with
`var _ interfaces.X = (*Impl)(nil)` next to the struct.

## Request flow

1. `middlewares.CORS → WithAuth → RequestLog` wrap the `http.ServeMux`.
2. `srv.BaseContext` seeds every request ctx once with `deps.Seed(modules.WithRegistry(ctx, ...), uses)`.
3. Handler is a plain `func(w, r)`; it reads deps with `deps.From(r.Context())`.
4. Handler resolves the engine family via `d.Modules.Engine(r.PathValue("engine"))`.
5. Handler calls a bound use case `Serve(ctx, ...)` inside `handle(w, r, fn)`, which opens a span and maps errors.
6. Use case reads repositories/modules from its own struct fields (filled by `deps.WithDeps` reflection), enforces access via `rawConn`/`Authorize`, persists via a repository.
7. `writeErr` maps `ErrNotFound`/`ErrUnsupported`→404, `ErrForbidden`→403, else 400.

## Conventions

- **One use case = one file = one `Serve` method.** No use case calls another; shared steps are package helpers in `usecases` (`raw.go`, `policy.go`, `apply.go`).
- **No env reads outside `internal/common/config`.** Add a `EnvX` const + field + `fieldOf` case.
- **No hardcoded drivers/families/statuses.** Add to `internal/constants`.
- **Secrets:** passwords are redacted on every response (`Connection.Redacted()`); encrypted at rest via `internal/common/secret` when `NOXIS_SECRET_KEY` set.
- **Authz:** the single gate is `usecases.rawConn` (per-connection) + `RequireAdmin` (management). Open mode / global admin bypass.
- **Errors:** wrap with `%w`; compare with `errors.Is`. Sentinel errors live in `interfaces` / `usecases`.
- **Spans:** `ctx, end := logging.StartSpan(ctx); defer func(){ end(err) }()` at the top of every repository/module/use case method.
- **No new dependency** for a few lines; no abstraction or option nobody asked for.
- **Migrations are append-only**: add to the `migrations` slice in `internal/repositories/sqlite/sqlite.go`; never edit an existing statement.
- **Changes are artifacts**: never delete a recorded change; a failed submission is returned, not stored.

## Notes that bite

- `Makefile swagger` rewrites `doc/swagger.yaml|json` from Swagger 2.0 → OpenAPI 3; the web client `gen:api` reads the same files. Regenerate and commit both after route/annotation changes.
- `cmd/noxis/server/handlers/static/` is gitignored except `placeholder.html`; `go:embed` needs a file at compile time.
- The retired master/slave + Kafka design is gone from the code, docs and CI. If an old branch, issue or note mentions `doorman`, `slave`, `kafka`, `NOXIS_BROKER` or `NOXIS_TOPIC_PREFIX`, it predates the rewrite — do not restore it.

## Docs index

- `doc/agent/architecture.md` — layers, DI, registry, patterns, how to extend.
- `doc/agent/infrastructure.md` — env vars, docker, CI/CD, local dev, build.
- `doc/agent/database.md` — SQLite schema, migrations, repositories, secrets.
- `doc/agent/review.md` — code-review checklist, invariants, definition of done.
- `doc/plan/` — one snapshot per `ai:y` plan (issue body + commit + PR) with an index; the plan of record for how the current tree got here.

## Open work (candidates for the next session)

- Confirm with the owner before starting; check open issues/PRs on `dimasbaguspm/noxis`.
- Known ceilings from the code: MongoDB writes have no validated rollback; memory-cache has no change queue (by design).
- No roadmap doc exists; the old `PLAN.md` phases (UCM, backup/ops) predate the rewrite. Derive the next roadmap from the README feature spec + the open ceilings, and write new plans in the issue format captured under `doc/plan/`.
