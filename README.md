# noXis

A lightweight console to query and change the databases you already run, with confidence.

**The goal:** point noXis at the databases you already run (Postgres,
MySQL/MariaDB, Redis, Valkey and MongoDB) and get a single place to browse them, run reads, and
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
- **Redis is not pretended into SQL** — a Redis value cannot be rolled back, so
  there is no validated change queue for it: keys can be browsed, inspected and
  removed, and every operation is a typed client call (no command text to
  inject). `FLUSHALL` is deliberately unreachable.
- **Namespaced API** — `/v1/sql/...`, `/v1/memcache/...` and `/v1/mongodb/...`;
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
  - **Dashboard** — every instance in one searchable table with live
    reachability (SSE) and a detail dialog; a swipeable tile counts All / SQL /
    Memory Cache and MongoDB; one "new connection" wizard asks for the kind of
    instance first, then verifies it before saving
  - menu (collapsible sidebar + a search field in the header that opens a
    searchable palette): Dashboard · SQL (Query, DML, DDL, Analytics) ·
    Memory Cache (Keys, Analytics) · MongoDB (Explore, Requests, Analytics)
  - **Query** — per-instance schema/table browser, monaco editor (self-hosted,
    schema-aware completion), result grid with duration / rows / plan estimate
    and CSV export; `ANALYZE` and "show records" live in each table's menu
  - **DML / DDL** — change queue, then a focus review page per change:
    estimates vs actuals, the tables/columns it touches (parsed from the
    statements), an activity trail, and monaco for query/rollback; changes are
    artifacts, never deleted
    - **Analytics** — live instance metrics over SSE (connections, cache hit
    ratio, throughput) with recharts graphs
  - **Memory Cache** — browse keys by pattern with type, TTL and size (one bounded
    scan, so the list can be sorted by TTL for real), inspect any value per its
    type, remove selected keys, and empty a database behind a typed
    confirmation; Analytics charts INFO (memory, clients, ops/sec, hit ratio,
    eviction)
  - **MongoDB** — Explore has three resizable panes (databases, collections,
    documents): pick a collection and filter it, reading the documents as
    **JSON, a grid or a tree**; Requests is the review queue plus a wizard
    (database → collection → filter and value, stacked) and a review page with
    the dry-run checks, the activity trail and no rollback editor; Analytics
    watches serverStatus, dbStats and collStats over SSE

## Docs

- `PLAN.md` — architecture, decisions, and known ceilings.
- `doc/swagger.yaml|json` — OpenAPI 3 spec (Postman artifact, generated from
  controller annotations via `make swagger`).
- `web/` — the console source (`make web` embeds it; `make dev-ui` gives a
  hot-reloading dev server on :5173).
- `test/uat/` — Gherkin acceptance suite over the real stack: `make uat`.

## Quick start

```bash
make dev          # noxis + a throwaway postgres to play with
open http://localhost:9000
# or drive the API:
curl -s -X POST localhost:9000/v1/sql/connections -H 'Content-Type: application/json' \
  -d '{"name":"demo","driver":"postgres","host":"db","port":5432,
       "user":"postgres","password":"noxis","database":"demo"}'
curl -s -X POST localhost:9000/v1/sql/connections/demo/query \
  -H 'Content-Type: application/json' -d '{"sql":"SELECT 6*6 AS n"}'

# mongodb: register the compose server, explore, run a request
curl -s -X POST localhost:9000/v1/mongodb/connections -H 'Content-Type: application/json' \
  -d '{"name":"docs","driver":"mongo","host":"mongo","port":27017,"user":"root",
       "password":"noxis","database":"admin"}'
curl -s "localhost:9000/v1/mongodb/connections/docs/collections?database=demo"
curl -s -X POST localhost:9000/v1/mongodb/connections/docs/requests \
  -H 'Content-Type: application/json' \
  -d '{"op":"find","database":"demo","collection":"items","filter":"{\"sku\": \"a\"}"}'

# memory cache: register the compose server, browse keys, read INFO
curl -s -X POST localhost:9000/v1/memcache/connections -H 'Content-Type: application/json' \
  -d '{"name":"cache","driver":"redis","host":"redis","port":6379,"database":"0"}'
curl -s "localhost:9000/v1/memcache/connections/cache/keys?pattern=*"
curl -s localhost:9000/v1/memcache/connections/cache/stats
```
