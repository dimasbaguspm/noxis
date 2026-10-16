# Infrastructure, build & dev workflow

Detail for `AGENTS.md`.

## Entry point & CLI

`apps/api/cmd/noxis/main.go` → `bootstrap.go` (`Bootstrap()`) builds the signal-aware
context, loads config, sets up logging, and runs one cobra root:

- `noxis run server` — REST API + embedded console (`apps/api/cmd/noxis/server.Run`).

There is no ticker process and no in-process scheduler. An external service (operator
cron) drives due changes by POSTing `/internal/callback/scheduler`, authenticated
with `NOXIS_SCHEDULER_API_KEY` + `NOXIS_SCHEDULER_SECRET` (bearer key + HMAC over
`<timestamp>.<body>`).

`server.Run` requires `NOXIS_LISTEN` and `NOXIS_DB`, opens the store, builds the
OIDC verifier (nil = open mode), assembles `deps.Uses`, seeds it on
`srv.BaseContext`, and starts the server with a 10s graceful shutdown.

## Environment variables (`apps/api/internal/common/config`)

| var | meaning |
| --- | --- |
| `NOXIS_LISTEN` | REST listen addr (default `:9000`) |
| `NOXIS_DB` | SQLite path (default `/data/noxis.db`) |
| `NOXIS_SECRET_KEY` | 64 hex chars / 32 bytes; **required** |
| `NOXIS_ADMIN_USERNAME` / `NOXIS_ADMIN_PASSWORD` | root local admin; required only while no accounts exist |
| `NOXIS_SCHEDULER_API_KEY` | external scheduler: shared api key (bearer) |
| `NOXIS_SCHEDULER_SECRET` | external scheduler: HMAC secret signing the trigger |
| `NOXIS_OTEL_COLLECTOR` | OTLP endpoint (optional) |
| `NOXIS_OTEL_NAME` | OTEL service name (optional) |
| `NOXIS_OIDC_ISSUER` | empty = OIDC off (local accounts still required); set = accept OIDC tokens too |
| `NOXIS_OIDC_CLIENT_ID` | public PKCE client id |
| `NOXIS_OIDC_AUDIENCE` | expected `aud`, defaults to client id |
| `NOXIS_OIDC_GROUPS_CLAIM` | defaults to `groups` |
| `NOXIS_OIDC_ADMIN_GROUP` | implicit global admin |

`config.LogState` logs secret/auth state once at boot. **No other package reads
the environment.** To add a var: `EnvX` const + `Config` field + `LoadConfig`
entry. Defaults and cross-field rules live in `LoadConfig` / `Validate`; a
`.env.example` at the repo root documents every flag.

## Makefile targets (all run from the repo root)

| target | what |
| --- | --- |
| `make build` | `cd apps/api && go build -o ../../bin/noxis ./cmd/noxis` |
| `make test` | `cd apps/api && go test ./...` |
| `make lint` | golangci-lint v2 in `apps/api` (installs on first use) |
| `make web` | pnpm install + build → sync `bin/dist/*` into the embed dir |
| `make swagger` | swag init + convert Swagger 2.0 → OpenAPI 3 into root `openapi.{yaml,json}` |
| `make dev` | compose stack (noxis + postgres/redis/valkey/mongo/mariadb) |
| `make dev-ui` | same + hot-reloading Vite console on :5173 |
| `make down` | stop compose and wipe volumes (`-v`) |
| `make uat` | godog acceptance suite over the real compose stack |

## Docker

Two images share `apps/api` as the Go build source:

- `Dockerfile` — full: builds `apps/web/` (node:22-alpine) → copies dist into
  `apps/api/cmd/noxis/server/handlers/static/` → golang:1.26-alpine build. This
  is what `docker-compose.yml` and `docker-publish.yml` use.
- `Dockerfile.api` — API only: no console build, serves the committed
  `placeholder.html`. `docker build -f Dockerfile.api -t noxis-api .`

Both end on alpine:3.20 with the static binary + `ca-certificates`.

`docker-compose.yml` at the repo root (`name: noxis`):

- `noxis` — the full image, `:9000`, named volume `noxis-data:/data`,
  healthcheck `/healthz`, waits on all target services healthy.
- `db` postgres:16, `redis` redis:7, `valkey` valkey/valkey:8, `mongo` mongo:7
  (auth on: root/noxis), `mariadb` mariadb:11.
- `console` — Vite dev server, `profiles: [dev]`, mounts `./apps/web`.

Secrets here are demo-only (`noxis`). Dev compose is a stand-in for target
databases that "already exist" in real use.

`deploy/docker-compose.yml` (`name: noxis`) is the **first-instance** stack:
noXis alone, one volume, no ticker. It requires `NOXIS_SECRET_KEY`,
`NOXIS_ADMIN_USERNAME` and `NOXIS_ADMIN_PASSWORD` (the `${VAR:?}` form fails
loudly when unset); `make up` runs it with `deploy/.env`. See the README "First
instance".

## Web console

`apps/web/` = React 18 + TS + Vite + Tailwind 4 + shadcn-style `components/ui`, data
via TanStack Query, monaco editor, `oidc-client-ts` for PKCE login.

- `pnpm dev` proxies `/v1` to `VITE_PROXY_TARGET` (default `http://localhost:9000`).
- `pnpm build` outputs to `<repo>/bin/dist`; `make web` syncs it into the Go embed.
- `pnpm gen:api` regenerates `src/api/generated.ts` from `openapi.yaml`.
- Layout: `src/pages/*` (routes), `src/components/*` (by domain: changes,
  connection, layout, mongodb, sql, ui), `src/hooks/*` (queries), `src/lib/*`.
- `apps/api/cmd/noxis/server/handlers/static/` is gitignored except `placeholder.html`
  (go:embed needs a compile-time file). `GetConsole` answers JSON 404 for any
  unknown `/v1/*` instead of serving the SPA shell.

## Devcontainer

`.devcontainer/` opens the repo in a ready environment:

- base `mcr.microsoft.com/devcontainers/go:1.26-bookworm` + Node 22 +
  Docker-in-Docker features;
- `post-create.sh` activates `pnpm@10.34.5` (from `apps/web/package.json`),
  runs `go mod download` in `apps/api` and `pnpm install` in `apps/web`;
- forwards `9000` (API + console) and `5173` (Vite).

`make dev` / `make dev-ui` work inside it because Docker-in-Docker provides the
compose daemon. Validate edits with a JSON parse and `bash -n post-create.sh`.

## CI/CD (`.github/workflows/`)

`ci.yml` — path-filtered:

- `changes` job runs `dorny/paths-filter`; heavy jobs run only for their side.
- `go` (if any `apps/api/** Makefile Dockerfile Dockerfile.api docker-compose.yml .github/workflows/**`): lint via `golangci/golangci-lint-action@v9` (pinned `v2.15.0`, prebuilt + cached) → both Docker builds through `docker/build-push-action` with GitHub Actions layer cache → unit tests (excluding `apps/api/test/uat`) → `make uat`.
- `web` (if `apps/web/**`, `openapi.*` or workflows): `cd apps/web && pnpm install --frozen-lockfile && pnpm lint && pnpm build`.

Never lint with `go run …/golangci-lint@latest`: it compiles the linter from source on every run (~3 min); the action installs a prebuilt binary in ~20s. Keep the version pinned in `.github/workflows/ci.yml` and `Makefile`.

`docker-publish.yml` — on push to `master` (or manual), builds the Dockerfile
and pushes `ghcr.io/<owner>/noxis:nightly`.

## Local workflow

1. `make test` and `make lint` (fast feedback).
2. `make dev` → `http://localhost:9000`; or `make dev-ui` for hot reload.
3. For a UI change: `make web` before `make build`/docker so the embed is fresh.
4. For a route change: `make swagger` and commit the regenerated `openapi.*`.
5. `make uat` before opening a PR when the REST surface changed.
