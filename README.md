# noxis

Your own multi-machine SQL database control plane.

**The goal:** manage SQL databases spread across your machines from one
place. Tell the master *which engines live on which node* — postgres,
mariadb, mysql — and that machine's worker builds and runs them. Then ask
questions, change schemas, or test changes safely, all through a single
REST API.

## What it does at a glance

- **One binary, two roles** — a `master` (REST API) and per-machine `slave`
  workers. Register a machine once; it converges itself from then on.
- **Event-driven over Kafka** (franz-go). Desired state flows master → slave;
  durable consumer offsets mean restarts resume cleanly — no reconnect loops,
  no extra broker daemons to babysit beyond Kafka itself.
- **SQL module (doorman)** — creates/updates engine instances (docker:
  image, persistent volume, isolated per-node network) and runs database ops
  with statement verification:
  - `query` — reads only (SELECT/SHOW/EXPLAIN/…) — any write rejected
  - `alter` — writes: DML (INSERT/UPDATE/…) and DDL (ALTER/CREATE/DROP/…)
  - `rollback` — dry-run inside a transaction that always rolls back
  - `analytics` — `ANALYZE` / `ANALYZE TABLE`
- **Engine updates that don't lose data** — same-major version bumps swap the
  container and keep the volume; major bumps get a fresh cluster with the old
  data preserved for manual migration.
- **Isolation by default** — each node's engines share one docker bridge
  network, invisible to other machines.
- **Small state footprint** — sqlite holds only the registry; specs and
  status live in Kafka + master memory.

## Docs

- `PLAN.md` — architecture, decisions, and known ceilings.
- `doc/swagger.yaml|json` — OpenAPI 3 spec (Postman artifact, generated from
  controller annotations via `make swagger`).
- `test/uat/` — Gherkin acceptance suite (godog): the real compose stack is
  driven through its public REST surface, `make uat` (skips @slow image-pull
  scenarios unless UAT_TAGS is set).
- `web/` — the React+TS operator console (`make web` builds it into the
  binary via embed; dev: `docker compose --profile dev up -d` → hot-reload on
  :5173, proxied to the master).