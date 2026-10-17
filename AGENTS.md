# AGENTS.md — noxis harness brief

Read this first in every session. It is the map; the four topic docs under
`docs/agent/` are the detail and the per-initiative history is under
`docs/plan/`. `README.md` is the authoritative current feature spec.

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
                 #   UAT_BUILD=1 rebuilds · UAT_KEEP=1 reuses the stack
```

CI (`.github/workflows/ci.yml`) runs exactly: lint → docker build → unit →
`make uat` for Go; lint + typecheck + build for `apps/web/`.

## Repo map (outer layer → inner layer)

Top level is the two apps plus infra:

| Path | Role |
| --- | --- |
| `apps/api/` | Go module `noxis`: REST API, use cases, engine modules, tests |
| `apps/web/` | React + TS + shadcn/ui console (Vite, pnpm) |
| `docs/agent/`, `docs/plan/` | agent docs + the `ai:y` plan archive |
| `.devcontainer/` | devcontainer: Go 1.26 + Node 22 + pnpm + Docker-in-Docker |
| `Makefile` | the single task entrypoint; delegates into `apps/*` |
| `Dockerfile` / `Dockerfile.api` | full image (web + api) / API-only image |
| `docker-compose.yml` | dev + UAT stack (noxis + target databases) |
| `openapi.json|yaml` | generated OpenAPI (Postman artifact) |

Inside `apps/api/`:

| Path | Role |
| --- | --- |
| `apps/api/cmd/noxis/` | process entry: CLI, server wiring (routes), middlewares |
| `apps/api/cmd/noxis/server/app.go` | composition root: every use case built explicitly |
| `apps/api/cmd/noxis/server/static/` | embedded console (built SPA; placeholder committed) |
| `apps/api/cmd/noxis/server/middlewares/` | CORS → auth → request-log/span |
| `apps/api/internal/usecases/` | root framework (RegistryDeps + helpers) + one subpackage per domain |
| `apps/api/internal/modules/` | engine adapters: `sql`, `memcache`, `mongodb`, `auth` + `Registry` |
| `apps/api/internal/repositories/` | persistence: `sqlite` (the only state) |
| `apps/api/internal/interfaces/` | seams (`ConnectionRepository`, `Module`, `SQLModule`, …) |
| `apps/api/internal/entity/` | shared data model (store + wire + REST) |
| `apps/api/internal/common/` | config, logging/span, idgen, pagination, secret |
| `apps/api/internal/constants/` | universal values (drivers, families, statuses, limits) |
| `apps/api/test/uat/` | Gherkin acceptance suite (`make uat`) |

Dependency direction is one-way: `handlers → usecases → modules → repositories → entity`.
`entity` imports nothing internal. Modules/repositories assert their seams with
`var _ interfaces.X = (*Impl)(nil)` next to the struct.

## Request flow

1. `middlewares.CORS → WithAuth → RequestLog` wrap the `http.ServeMux`.
2. `newHandler(app, cfg)` registers each route straight at a use case `Serve(w, r)`.
3. `Serve` reads the path/query/body and resolves the engine family via `uc.Modules.Engine(...)`.
4. It calls its `Run(ctx, ...)` domain core and writes the response with `respond`/`fail`.
5. The use case reads repositories/modules from its own struct fields (set explicitly in `newApp`), enforces access via `rawConn`/`Authorize`, persists via a repository.
6. `fail` maps `constants.ErrNotFound`/`constants.ErrUnsupported`→404, `constants.ErrForbidden`→403, `constants.ErrInvalidCredentials`/`constants.ErrUnauthorized`→401, else 400.

## Conventions

- **One use case = one file = one `Serve` method.** No use case calls another; shared steps are package helpers in `usecases` (`raw.go`, `policy.go`, `apply.go`).
- **No env reads outside `apps/api/internal/common/config`.** Add a `EnvX` const + field + `fieldOf` case.
- **No hardcoded drivers/families/statuses.** Add to `apps/api/internal/constants`.
- **Secrets:** passwords are redacted on every response (`Connection.Redacted()`); encrypted at rest via `apps/api/internal/common/secret` when `NOXIS_SECRET_KEY` set.
- **Authz:** the single gate is `usecases.rawConn` (per-connection) + `RequireAdmin` (management). Open mode / global admin bypass.
- **Errors:** wrap with `%w`; compare with `errors.Is`. Sentinel errors live in `interfaces` / `usecases`.
- **Spans:** `ctx, end := logging.StartSpan(ctx); defer func(){ end(err) }()` at the top of every repository/module/use case method.
- **No new dependency** for a few lines; no abstraction or option nobody asked for.
- **Migrations are append-only**: add to the `migrations` slice in `apps/api/internal/repositories/sqlite/sqlite.go`; never edit an existing statement.
- **Changes are artifacts**: never delete a recorded change; a failed submission is returned, not stored.

## Notes that bite

- `Makefile swagger` rewrites `openapi.yaml|json` from Swagger 2.0 → OpenAPI 3; the web client `gen:api` reads the same files. Regenerate and commit both after route/annotation changes.
- `apps/api/cmd/noxis/server/static/` is gitignored except `placeholder.html`; `go:embed` needs a file at compile time.
- The retired master/slave + Kafka design is gone from the code, docs and CI. If an old branch, issue or note mentions `doorman`, `slave`, `kafka`, `NOXIS_BROKER` or `NOXIS_TOPIC_PREFIX`, it predates the rewrite — do not restore it.

## Docs index

- `docs/agent/architecture.md` — layers, DI, registry, patterns, how to extend.
- `docs/agent/infrastructure.md` — env vars, docker, CI/CD, local dev, build.
- `docs/agent/database.md` — SQLite schema, migrations, repositories, secrets.
- `docs/agent/review.md` — code-review checklist, invariants, definition of done.
- `docs/plan/` — one snapshot per `ai:y` plan (issue body + commit + PR) with an index; the plan of record for how the current tree got here.

## Open work (candidates for the next session)

- Confirm with the owner before starting; check open issues/PRs on `dimasbaguspm/noxis`.
- Known ceilings from the code: MongoDB writes have no validated rollback; memory-cache has no change queue (by design).
- No roadmap doc exists; the old `PLAN.md` phases (UCM, backup/ops) predate the rewrite. Derive the next roadmap from the README feature spec + the open ceilings, and write new plans in the issue format captured under `docs/plan/`.
