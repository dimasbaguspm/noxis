# noXis

A lightweight console to query and change the databases you already run, with confidence.

**The goal:** point noXis at the databases you already run (Postgres,
MySQL/MariaDB, SQLite, Redis, Valkey and MongoDB) and get a single place to
browse them, run reads, and
ship changes safely. No agents, no brokers, no orchestration — one binary, one
volume, no docker socket.

## What it does at a glance

- **Connections** — register a database (host/port/user/password/database);
  passwords never leave the API (redacted on every response).
- **Browse** — introspect schema (tables + columns) and preview table rows.
- **Reads run directly** — `SELECT`/`EXPLAIN`/`SHOW` execute immediately with a
  result grid; the read path is read-only and rejects DML/DDL.
- **Changes are a workflow** — submit *{connection, notes, query, rollback}*:
  - **validate** — the query and its rollback are dry-run **together** in a
    transaction that always rolls back, proving the rollback really undoes the
    change (nothing persists on submit);
  - **approve** — self-approval is fine (single-operator tool);
  - **apply** — executed inside a transaction; failures stay recorded on the
    change.
- **Honest about drivers** — MySQL commits DDL implicitly, so a change carrying
  DDL there is marked `unverified` instead of pretending a dry-run happened.
- **SQLite is a file** — a SQLite connection is a path on the noXis
  host/container (mount it), not a server. Its changes dry-run in a transaction
  that always rolls back, since SQLite DDL is transactional. Point it at a
  database file — never noXis's own registry.
- **Redis is not pretended into SQL** — a Redis value cannot be rolled back, so
  there is no validated change queue for it: keys can be browsed, inspected and
  removed, and every operation is a typed client call (no command text to
  inject). `FLUSHALL` is deliberately unreachable.
- **Namespaced API** — `/external/v1/engine/sql/...`, `/external/v1/engine/memcache/...` and `/external/v1/engine/mongodb/...`;
  each family serves only its own drivers, so a memory-cache connection is
  invisible to the SQL routes and vice versa. The console follows the same
  split: one dashboard table for every instance with a detail dialog, and a
  sidebar group per family.
- **Redis and Valkey are one family: Memory Cache** — Valkey implements the
  Redis protocol and command set, so it is served by the same module under its
  own driver name (`valkey`), and INFO is read from its own `valkey_version` so
  the console shows what it is really talking to. The acceptance suite
  registers a real Valkey server to keep that claim honest.
- **Operator console** — served by the binary itself at `/` (React + TS +
  shadcn/ui, embedded; no build step at deploy time):
  - **Dashboard** — every instance in one searchable table with a detail
    dialog; a swipeable tile counts All / SQL / Memory Cache and MongoDB; one
    "new connection" wizard asks for the kind of instance first, then verifies
    it before saving
  - menu (collapsible sidebar + a search field in the header that opens a
    searchable palette): Dashboard · SQL (Query, DML, DDL) ·
    Memory Cache (Keys) · MongoDB (Explore, Requests)
  - **Query** — per-instance schema/table browser, monaco editor (self-hosted,
    schema-aware completion), result grid with duration / rows / plan estimate
    and CSV export; `ANALYZE` and "show records" live in each table's menu
  - **DML / DDL** — change queue, then a focus review page per change:
    estimates vs actuals, the tables/columns it touches (parsed from the
    statements), an activity trail, and monaco for query/rollback; changes are
    artifacts, never deleted
  - **Memory Cache** — browse keys by pattern with type, TTL and size (one bounded
    scan, so the list can be sorted by TTL for real), inspect any value per its
    type, remove selected keys, and empty a database behind a typed
    confirmation
  - **MongoDB** — Explore has three resizable panes (databases, collections,
    documents): pick a collection and filter it, reading the documents as
    **JSON, a grid or a tree**; Requests is the review queue plus a wizard
    (database → collection → filter and value, stacked) and a review page with
    the dry-run checks, the activity trail and no rollback editor

## Authentication

Noxis requires a login: every `/external/v1/*` call needs a bearer token. There is no
open mode. On first boot the root admin is created from `NOXIS_ADMIN_USERNAME`
and `NOXIS_ADMIN_PASSWORD` (required only while no accounts exist; ignored
after, so a password changed in the console survives restarts).

Accounts are local by default: `POST /external/v1/auth/login` verifies a bcrypt-hashed
password and returns a single opaque session token; disabling an account or
changing its password invalidates that token immediately.

OIDC can be added later and runs alongside local accounts — set
`NOXIS_OIDC_ISSUER` and noxis accepts the issuer's tokens too:

| env | meaning |
| --- | --- |
| `NOXIS_OIDC_ISSUER` | issuer URL; empty keeps OIDC off |
| `NOXIS_OIDC_CLIENT_ID` | public client id (PKCE, no secret) |
| `NOXIS_OIDC_AUDIENCE` | expected `aud`, defaults to the client id |
| `NOXIS_OIDC_GROUPS_CLAIM` | token claim holding group names, defaults to `groups` |
| `NOXIS_OIDC_ADMIN_GROUP` | members are global admins (manage connections and permissions) |

Access is per connection — `read` < `write` < `admin` — granted to a user (by
subject) or a group through `/external/v1/engine/{engine}/connections/{name}/permissions`.
Default deny: a user with no permission sees nothing. `GET /external/v1/me` reports the
caller and its effective level per connection (`GET /external/v1/auth/config` is the public
console bootstrap).

## Credentials at rest

Connection passwords cannot be hashed — they must be sent to the database — so
set `NOXIS_SECRET_KEY` (64 hex characters, e.g. `openssl rand -hex 32`) to
encrypt them in SQLite with AES-256-GCM. It is required: the server refuses to
start without it. Setting the key re-encrypts existing plaintext rows when the
store opens. Rotate a password from the instance's detail page.

## Docs

- `AGENTS.md` — the harness brief: repo map, conventions, drift, next steps.
- `docs/agent/` — architecture, infrastructure, database and code-review docs.
- `docs/plan/` — one snapshot per `ai:y` plan (issue body + commit + PR), with an index.
- `openapi.yaml|json` — OpenAPI 3 spec (Postman artifact, generated from
  handler annotations via `make swagger`).
- `apps/web/` — the console source (`make web` embeds it; `make dev-ui` gives a
  hot-reloading dev server on :5173).
- `apps/api/test/uat/` — Gherkin acceptance suite over the real stack: `make uat`.

## Quick start

```bash
make dev          # noxis + a throwaway postgres to play with
open http://localhost:9000

# the dev stack seeds the root admin admin / noxis-admin; every /v1 call needs
# the bearer token it returns (the console login form arrives in a later change)
TOKEN=$(curl -s localhost:9000/external/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"noxis-admin"}' | sed 's/.*"token":"\([^"]*\)".*/\1/')

# or drive the API (add -H "Authorization: Bearer $TOKEN" to each call):
curl -s -X POST localhost:9000/external/v1/engine/sql/connections -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"demo","driver":"postgres","host":"db","port":5432,
       "user":"postgres","password":"noxis","database":"demo"}'
curl -s -X POST localhost:9000/external/v1/engine/sql/connections/demo/query \
  -H 'Content-Type: application/json' -d '{"sql":"SELECT 6*6 AS n"}'

# mongodb: register the compose server, explore, run a request
curl -s -X POST localhost:9000/external/v1/engine/mongodb/connections -H 'Content-Type: application/json' \
  -d '{"name":"docs","driver":"mongo","host":"mongo","port":27017,"user":"root",
       "password":"noxis","database":"admin"}'
curl -s "localhost:9000/external/v1/engine/mongodb/connections/docs/collections?database=demo"
curl -s -X POST localhost:9000/external/v1/engine/mongodb/connections/docs/requests \
  -H 'Content-Type: application/json' \
  -d '{"op":"find","database":"demo","collection":"items","filter":"{\"sku\": \"a\"}"}'

# memory cache: register the compose server, browse keys
curl -s -X POST localhost:9000/external/v1/engine/memcache/connections -H 'Content-Type: application/json' \
  -d '{"name":"cache","driver":"redis","host":"redis","port":6379,"database":"0"}'
curl -s "localhost:9000/external/v1/engine/memcache/connections/cache/keys?pattern=*"
```
