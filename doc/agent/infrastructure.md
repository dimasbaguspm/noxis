# Infrastructure, build & dev workflow

Detail for `AGENTS.md`.

## Entry point & CLI

`cmd/noxis/main.go` → `bootstrap.go` (`Bootstrap()`) builds the signal-aware
context, loads config, sets up logging, and runs one cobra root:

- `noxis run server` — REST API + embedded console (`cmd/noxis/server.Run`).
- `noxis run ticker` — standalone scheduler: POSTs
  `/internal/callback/scheduler` on `NOXIS_TICK_INTERVAL`, requires
  `NOXIS_TICK_HOST`. The server carries no ticker.

`server.Run` requires `NOXIS_LISTEN` and `NOXIS_DB`, opens the store, builds the
OIDC verifier (nil = open mode), assembles `deps.Uses`, seeds it on
`srv.BaseContext`, and starts the server with a 10s graceful shutdown.

## Environment variables (`internal/common/config`)

| var | meaning |
| --- | --- |
| `NOXIS_LISTEN` | REST listen addr (**required**) |
| `NOXIS_DB` | SQLite path (**required**) |
| `NOXIS_TICK_HOST` | ticker only: server base URL (**required** for ticker) |
| `NOXIS_TICK_INTERVAL` | ticker only: e.g. `30s` (default 30s) |
| `NOXIS_SECRET_KEY` | 64 hex chars / 32 bytes; empty = plaintext passwords |
| `NOXIS_OTEL_COLLECTOR` | OTLP endpoint (optional) |
| `NOXIS_OTEL_NAME` | OTEL service name (optional) |
| `NOXIS_OIDC_ISSUER` | empty = open API; set = authenticate every `/v1/*` |
| `NOXIS_OIDC_CLIENT_ID` | public PKCE client id |
| `NOXIS_OIDC_AUDIENCE` | expected `aud`, defaults to client id |
| `NOXIS_OIDC_GROUPS_CLAIM` | defaults to `groups` |
| `NOXIS_OIDC_ADMIN_GROUP` | implicit global admin |

`config.LogState` logs secret/auth state once at boot. **No other package reads
the environment.** To add a var: `EnvX` const + `Config` field + `LoadConfig` +
`fieldOf` case (+ `Require(...)` if mandatory).

## Makefile targets

| target | what |
| --- | --- |
| `make build` | `go build -o bin/noxis ./cmd/noxis` |
| `make test` | `go test ./...` |
| `make lint` | golangci-lint v2 (installs on first use) |
| `make web` | pnpm install + build → sync `bin/dist/*` into the embed dir |
| `make swagger` | swag init + convert Swagger 2.0 → OpenAPI 3 into `doc/` |
| `make dev` | compose stack (noxis + postgres/redis/valkey/mongo/mariadb) |
| `make dev-ui` | same + hot-reloading Vite console on :5173 |
| `make down` | stop compose and wipe volumes (`-v`) |
| `make uat` | godog acceptance suite over the real compose stack |

## Docker

`Dockerfile` is multi-stage:

1. `web` (node:22-alpine): install pnpm, build `web/` → `/bin/dist`.
2. `build` (golang:1.26-alpine): copy dist into
   `cmd/noxis/server/handlers/static/`, `CGO_ENABLED=0 go build`.
3. runtime (alpine:3.20): binary + `ca-certificates`.

`docker/docker-compose.yml` (`name: noxis`):

- `noxis` — the app, `:9000`, named volume `noxis-data:/data`, healthcheck
  `/healthz`, waits on all target services healthy.
- `ticker` — same image, `run ticker`, depends on noxis healthy.
- `db` postgres:16, `redis` redis:7, `valkey` valkey/valkey:8, `mongo` mongo:7
  (auth on: root/noxis), `mariadb` mariadb:11.
- `console` — Vite dev server, `profiles: [dev]`.

Secrets here are demo-only (`noxis`). Dev compose is a stand-in for target
databases that "already exist" in real use.

## Web console

`web/` = React 18 + TS + Vite + Tailwind 4 + shadcn-style `components/ui`, data
via TanStack Query, monaco editor, `oidc-client-ts` for PKCE login.

- `pnpm dev` proxies `/v1` to `VITE_PROXY_TARGET` (default `http://localhost:9000`).
- `pnpm build` outputs to `../bin/dist`; `make web` syncs it into the Go embed.
- `pnpm gen:api` regenerates `src/api/generated.ts` from `doc/swagger.yaml`.
- Layout: `src/pages/*` (routes), `src/components/*` (by domain: changes,
  connection, layout, mongodb, sql, ui), `src/hooks/*` (queries), `src/lib/*`.
- `cmd/noxis/server/handlers/static/` is gitignored except `placeholder.html`
  (go:embed needs a compile-time file). `GetConsole` answers JSON 404 for any
  unknown `/v1/*` instead of serving the SPA shell.

## CI/CD (`.github/workflows/`)

`ci.yml` — path-filtered:

- `changes` job runs `dorny/paths-filter`; heavy jobs run only for their side.
- `go` (if any `cmd/** internal/** test/** go.mod go.sum .golangci.yml Makefile Dockerfile docker/**`): lint → docker build → unit tests (excluding `test/uat`) → `make uat`.
- `web` (if `web/**` or workflows): `pnpm install --frozen-lockfile && pnpm lint && pnpm build`.

`docker-publish.yml` — on push to `master` (or manual), builds the Dockerfile
and pushes `ghcr.io/<owner>/noxis:nightly`.

## Local workflow

1. `make test` and `make lint` (fast feedback).
2. `make dev` → `http://localhost:9000`; or `make dev-ui` for hot reload.
3. For a UI change: `make web` before `make build`/docker so the embed is fresh.
4. For a route change: `make swagger` and commit the regenerated `doc/swagger.*`.
5. `make uat` before opening a PR when the REST surface changed.
