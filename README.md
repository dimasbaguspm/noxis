# noXis

A lightweight console to query and change the databases you already run, with confidence.

**The goal:** point noxis at your databases (Postgres, MySQL/MariaDB) and get a
single place to browse them, run reads, and ship changes safely. No agents, no
brokers, no orchestration — one binary, one volume, no docker socket.

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
- **Operator console** — served by the binary itself at `/` (React + TS +
  shadcn/ui, embedded; no build step at deploy time):
  - **Dashboard** — registered instances in a searchable table with live
    reachability (SSE), executed DDL/DML counts and the error rate; add an
    instance (verified before it is saved)
  - menu (sidebar + a searchable dropdown in the header): Dashboard ·
    SQL (Query, DML, DDL, Analytics) · MongoDB and Redis (scoped, pages
    describe the plan)
  - **Query** — per-instance schema/table browser, monaco editor (self-hosted,
    schema-aware completion), result grid with duration / rows / plan estimate
    and CSV export; `ANALYZE` and "show records" live in each table's menu
  - **DML / DDL** — change queue, then a focus review page per change:
    estimates vs actuals, the tables/columns it touches (parsed from the
    statements), an activity trail, and monaco for query/rollback; changes are
    artifacts, never deleted
  - **Analytics** — live instance metrics over SSE (connections, cache hit
    ratio, throughput) with recharts graphs

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
```
