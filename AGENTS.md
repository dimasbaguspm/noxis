# AGENTS.md — noxis harness brief

Read this first in every session. It is the map; the four topic docs under
`docs/agent/` are the detail and the per-initiative history is under
`docs/plan/`. `README.md` is the authoritative current feature spec.

## What this repo is

`noxis` = one Go binary + embedded React console. Point it at databases you
already run (Postgres, MySQL/MariaDB, Redis, Valkey, MongoDB), browse them,
run reads directly, and ship writes through a reviewed change workflow. No
agents, no Kafka, no docker socket. State lives in one SQLite file.

## Workflow (epic → tickets → one MR per ticket)

Every change lands through the tracker, never as an uncommitted blob on `master`:

1. **Epic** — open one plan issue describing the outcome. Its body follows the
   repo template (`.github/ISSUE_TEMPLATE/ticket.md`):
   `What problem does this solve? → What is the user facing? → How will I solve
   it? → Done when → Risks & notes`. Write one line per paragraph; never
   hard-wrap.
2. **Tickets** — split the epic into issues, one vertical slice each (cuts every
   layer, demoable alone, finishes in a session). Load the `ticket` skill for the
   title/body format; show the draft list for approval before creating.
3. **Branch + MR per ticket** — one branch and one PR per issue, never a second
   ticket's work in the same PR. Load the `mr` skill; title
   `[ai:y][#<issue>] <type>: <subject>`, body follows
   `.github/PULL_REQUEST_TEMPLATE.md` and closes the issue.
4. **Review before merge** — the human reviews the MR against its ticket's
   `Done when` checklist. `make lint`/`make test`/`make uat` and the web checks
   pass in CI before merge.
5. **Record** — snapshot the completed plan into `docs/plan/<id>-<slug>.md` and
   add it to the index, per `docs/plan/README.md`.

Do not start implementation until the epic and its tickets exist.

## Next action (verify the tree before touching it)

```bash
make test        # go test ./...            (~30s)
make lint        # golangci-lint v2         (~60s)
make build       # bin/noxis
make web         # build console into embed dir (needs pnpm)
make mocks       # regenerate internal/mocks from internal/interfaces (go.uber.org/mock)
make uat         # godog over real compose stack (needs docker, tags ~@slow)
                 #   UAT_BUILD=1 rebuilds · UAT_KEEP=1 reuses the stack
```

CI (`.github/workflows/ci.yml`) runs exactly: lint → api image build → unit →
`make uat` (which builds the full image through compose) for Go; lint +
typecheck + build for `apps/web/`.

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
| `apps/api/cmd/noxis/server/wiring.go` | composition root: the framework, the registry and the auth service, built once |
| `apps/api/cmd/noxis/server/static/` | embedded console (built SPA; placeholder committed) |
| `apps/api/cmd/noxis/server/middlewares/` | CORS → auth → request-log/span |
| `apps/api/internal/usecases/` | root framework (RegistryDeps + helpers) + one subpackage per domain |
| `apps/api/internal/modules/` | engine adapters: one package per driver (`postgres`, `mysql`, `mariadb`, `sqlite`, `cockroach`, `redis`, `valkey`, `mongodb`) + `Registry` |
| `apps/api/internal/repositories/` | persistence: `sqlite` (the only state) |
| `apps/api/internal/interfaces/` | seams (`ConnectionRepository`, `Module` + one capability interface per verb set, …) |
| `apps/api/internal/entity/` | shared data model (store + wire + REST) |
| `apps/api/internal/common/` | config, logging/span, idgen, pagination, secret, `sqltext` (the pure SQL lexer every engine calls), `auth` (the sign-in and OIDC service) |
| `apps/api/internal/constants/` | universal values (drivers, capabilities, statuses, limits) |
| `apps/api/test/uat/` | Gherkin acceptance suite (`make uat`) |

Dependency direction is one-way: `handlers → usecases → modules → repositories → entity`.
`entity` imports only `constants`: the change state machine lives on the change it governs. Modules/repositories assert their seams with
`var _ interfaces.X = (*Impl)(nil)` next to the struct.

## Request flow

1. `middlewares.CORS → Trace → WithAuth → RequestLog` wrap the `http.ServeMux`: the span covers the whole request (a refused token is traced), and the request log runs after the principal is attached.
2. `routes(app, cfg)` lists every endpoint as `{Method, Endpoint, Handler, Public, Middlewares}`; `newHandler` registers the table and attaches auth per route.
3. `Serve` reads the path/query/body and resolves one capability for the driver the route names with `uc.RouteModule(ctx, r, &target)`; the loader asserts the interface against that driver's module. A driver that cannot do the thing is refused by name, never stubbed.
4. It calls its `Run(ctx, ...)` domain core and writes the response with `respond`/`fail`.
5. The use case reads repositories/modules from its own struct fields (set once in `cmd/noxis/server/wiring.go`), enforces access via `RawConn`/`Authorize`, persists via a repository.
6. `fail` maps `constants.ErrNotFound`/`constants.ErrUnsupported`→404, `constants.ErrForbidden`→403, `constants.ErrInvalidCredentials`/`constants.ErrUnauthorized`→401, else 400.

## Conventions

- **One use case = one file = one `Serve` method.** No use case calls another; shared steps are package helpers in `usecases` (`raw.go`, `policy.go`, `changes.go`, `run.go`).
- **Nothing is shared across modules except a pure function, and pure functions live in `common`.** A module proves and applies; the change lifecycle, the queue and the HTTP preamble are orchestration and live in `usecases`.
- **Context:** authenticated principal and immutable module loader only. Repositories and other app dependencies stay explicit; use `common.LoadModule(ctx, &target)` for registered typed modules.
- **Repositories:** pass one `repositories.Registry` through app sources and expose it as `RegistryDeps.Repositories`; modules receive only the repositories they use.
- **No env reads outside `apps/api/internal/common/config`.** Add a `EnvX` const + field + `fieldOf` case.
- **No hardcoded drivers/capabilities/statuses.** Add to `apps/api/internal/constants`.
- **Secrets:** passwords are redacted on every response (`Connection.Redacted()`); encrypted at rest via `apps/api/internal/common/secret` when `NOXIS_SECRET_KEY` set.
- **Authz:** the single gate is `usecases.RawConn` (per-connection) + `RequireSuper` (global management). The env root is the only super; everyone else needs a non-expired per-connection grant.
- **Errors:** wrap with `%w`; compare with `errors.Is`. Sentinel errors live in `interfaces` / `usecases`.
- **Spans:** `ctx, end := logging.StartSpan(ctx); defer func(){ end(err) }()` at the top of every repository/module/use case method.
- **No new dependency** for a few lines; no abstraction or option nobody asked for.
- **Console state lives in a hook.** A React component in `apps/web` with three or
  more `useState` calls moves that state into a hook under `apps/web/src/hooks/`
  and keeps only JSX and event wiring. Two is the ceiling; a third is a review
  blocker, not a style preference. The vendored primitives in
  `apps/web/src/components/ui/` are exempt.
- **Test doubles are generated**: `apps/api/internal/mocks` comes from
  `internal/interfaces` via `make mocks`. Regenerate it whenever an interface in
  that package changes shape, and commit the result — a stale mock fails to
  compile or lies about the seam. A use-case test states its own expectations;
  real SQLite covers persistence.
- **SQLite schema:** `schemaTemplate` is canonical and versioned with `PRAGMA user_version`; no runtime migrations/backfills. Old or mismatched stores fail with a reset instruction; never silently drop them.
- **Changes are artifacts**: never delete a recorded change; a failed submission is returned, not stored.

## Notes that bite

- `Makefile swagger` rewrites `openapi.yaml|json` from Swagger 2.0 → OpenAPI 3; the web client `gen:api` reads the same files. Regenerate and commit both after route/annotation changes.
- `apps/api/cmd/noxis/server/static/` is gitignored except `placeholder.html`; `go:embed` needs a file at compile time.
- The retired master/slave + Kafka design is gone from the code, docs and CI. If an old branch, issue or note mentions `doorman`, `slave`, `kafka`, `NOXIS_BROKER` or `NOXIS_TOPIC_PREFIX`, it predates the rewrite — do not restore it.

## Docs index

- `docs/agent/architecture.md` — layers, DI, registry, patterns, how to extend.
- `docs/agent/infrastructure.md` — env vars, docker, CI/CD, local dev, build.
- `docs/agent/database.md` — SQLite schema lifecycle, repositories, secrets.
- `docs/agent/review.md` — code-review checklist, invariants, definition of done.
- `docs/plan/` — one snapshot per `ai:y` plan (issue body + commit + PR) with an index; the plan of record for how the current tree got here.

## Open work (candidates for the next session)

- Confirm with the owner before starting; check open issues/PRs on `dimasbaguspm/noxis`.
- Known ceilings from the code: MongoDB writes have no validated rollback; memory-cache has no change queue (by design).
- No roadmap doc exists; the old `PLAN.md` phases (UCM, backup/ops) predate the rewrite. Derive the next roadmap from the README feature spec + the open ceilings, and write new plans in the issue format captured under `docs/plan/`.
