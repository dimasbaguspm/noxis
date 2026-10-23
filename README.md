<p align="center">
  <img src="docs/assets/logo.webp" width="120" height="120" alt="noXis">
</p>

# noXis

**The calm way to change the databases you already run.**

noXis is a small, self-hosted console for Postgres, MySQL/MariaDB, SQLite,
Redis, Valkey and MongoDB. Point it at what you already run, browse freely, run
reads instantly, and send schema or data changes through a review that proves
the rollback **before** anyone approves.

One Go binary. One SQLite file. No agents, no brokers, no docker socket.

[Quick start](#quick-start) · [Why noXis](#why-noxis) · [What you get](#what-you-get) · [How a change works](#how-a-change-works) · [Docs](#docs)

## Why noXis

You already run the databases. The hard part is *changing* them.

- “Just run this one query” — at 11pm, in production.
- A migration that looked fine right up until it wasn’t.
- A “simple” admin tool that wants an agent, a broker, or root on your host
  just to show you a table.

noXis is the small thing in the middle. Look around as much as you like. When
it’s time to change something, noXis shows you the blast radius, checks that the
rollback actually undoes the change, and keeps the whole story afterward.

## What you get

- **Browse anything you run** — connections, schemas, tables and rows across SQL,
  memory cache and document databases, from one console.
- **Reads are instant** — `SELECT`, `EXPLAIN` and `SHOW` run straight away. The
  read path is read-only and refuses to become a way to change data.
- **Writes are a decision** — a change is `{connection, notes, query, rollback}`.
  It is dry-run **together with its rollback** in a transaction that always rolls
  back, so you see exactly what it touches and know the undo works.
- **Review that lasts** — every change is an artifact: its checks, estimates
  versus actuals, the tables it touched, and an activity trail. Nothing is
  silently deleted; a rejected submission is returned, not stored.
- **Small on purpose** — one binary, one volume, one SQLite file. Your databases
  stay yours; noXis never asks for a sidecar.
- **Secrets stay secret** — connection passwords are encrypted at rest
  (AES-256-GCM) and redacted from every response.
- **Honest about limits** — where a rollback genuinely cannot be proven, noXis
  says so instead of pretending. More [below](#honest-about-limits).

## How a change works

1. **Submit** — the query, a short note, and the rollback you’d run.
2. **Validate** — query and rollback run together in a transaction that is
   always rolled back. The estimate, the affected tables and the checks come back
   with the change.
3. **Approve** — self-approval is fine; this is a single-operator tool.
4. **Apply** — executed in a transaction. Failures stay recorded with their
   reason, so the next person knows what happened.

## Quick start

```bash
docker run -d --name noxis -p 9000:9000 \
  -v noxis-data:/data \
  -e NOXIS_SECRET_KEY="$(openssl rand -hex 32)" \
  -e NOXIS_ADMIN_EMAIL=root@example.com \
  -e NOXIS_ADMIN_PASSWORD=change-me \
  ghcr.io/dimasbaguspm/noxis:nightly
```

Open <http://localhost:9000>, sign in, and add your first connection.

Working on noXis itself, or want a throwaway stack to play with?

```bash
make dev        # noxis + a demo postgres, redis, valkey, mongo and mariadb
```

The dev stack seeds the root super admin (`root@noxis.local` / `noxis-admin`).

### Drive it from the API

Everything the console does is a REST call under `/external/v1/engine/{sql|memcache|mongodb}`.
Every call needs the bearer token from `POST /external/v1/auth/login`:

```bash
TOKEN=$(curl -s localhost:9000/external/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"root@noxis.local","password":"noxis-admin"}' | jq -r .token)

curl -s -X POST localhost:9000/external/v1/engine/sql/connections/demo/query \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"sql":"SELECT 6*6 AS n"}'
```

The full OpenAPI spec is in [`openapi.yaml`](openapi.yaml).

## Supported databases

| You run | noXis gives you |
| --- | --- |
| **Postgres** | browse, reads, validated changes, scratch-schema dry runs |
| **MySQL / MariaDB** | browse, reads, changes (see the DDL note below) |
| **SQLite** | browse, reads, transactional dry runs against a file |
| **Redis / Valkey** | browse keys by pattern, inspect values, TTL/size, remove keys |
| **MongoDB** | explore databases/collections/documents, reviewed write requests |

Redis and Valkey are treated as one family under their own driver names, because
Valkey speaks the same protocol — and the acceptance suite registers a real
Valkey server to keep that claim honest.

## Honest about limits

- **MySQL commits DDL implicitly.** A change carrying DDL there is marked
  `unverified` rather than pretending a dry run happened.
- **A cache value cannot be rolled back.** So memory cache has no change queue:
  you can browse, inspect and remove keys, and `FLUSHALL` is deliberately
  unreachable.
- **SQLite is a file.** Point the connection at the database file on the noXis
  host or volume — never at noXis’s own registry.

## Authentication

Every `/external/v1/*` call needs a bearer token; there is no open mode. Local
accounts sign in with an email and password (bcrypt, one opaque session token
each); the env root is the account created on first boot from
`NOXIS_ADMIN_EMAIL` + `NOXIS_ADMIN_PASSWORD`. OIDC can run alongside them
(`NOXIS_OIDC_ISSUER`, `NOXIS_OIDC_CLIENT_ID`, …).

Upgrading from `NOXIS_ADMIN_USERNAME`: set `NOXIS_ADMIN_EMAIL` and, once,
`NOXIS_ADMIN_PASSWORD`; on an existing store the new root is created and the old
root account is demoted.

Access is per connection — `read` < `write` < `admin` — granted to users or
groups, default deny.

Connection passwords are encrypted at rest with `NOXIS_SECRET_KEY` (required;
`openssl rand -hex 32`).

## Docs

- [`AGENTS.md`](AGENTS.md) — the harness brief: repo map, conventions, next steps.
- [`docs/agent/`](docs/agent) — architecture, infrastructure, database and review docs.
- [`docs/plan/`](docs/plan) — one snapshot per shipped plan.
- [`openapi.yaml`](openapi.yaml) · [`openapi.json`](openapi.json) — the REST contract.
- [`apps/web/`](apps/web) — the console source (`make web` embeds it).
- [`apps/api/test/uat/`](apps/api/test/uat) — the Gherkin acceptance suite (`make uat`).
