# noXis

**Reviewed database changes for agents and the humans who answer for them.**

[![ci](https://github.com/dimasbaguspm/noxis/actions/workflows/ci.yml/badge.svg)](https://github.com/dimasbaguspm/noxis/actions/workflows/ci.yml)

An agent reads your MySQL or MariaDB through one API. When something has to change, it writes the
statement *and* the statement that undoes it, and gets back a rehearsal: the schema rebuilt in a
throwaway database, the diff, the rows it touches, the locks it takes. A human holding a different
token decides. Then it runs, and the receipt is kept forever.

Nothing is installed beside your database. One binary, one state file, one API contract.

---

## The problem

Giving an agent a database credential is easy. Giving it one you can defend in an incident review is
not. Everything available today picks one of three bad positions:

- **A read-only user.** Safe and useless: the moment the agent finds a missing index or a wrong column
  type, a human is back to hand-writing DDL from a chat transcript.
- **A write-capable user.** The agent becomes the reviewer of its own work. `DROP COLUMN` on a table
  with 40 million rows is one typo away, and nobody can say afterwards who asked for what.
- **A dry run.** `BEGIN; …; ROLLBACK;` is not evidence for DDL. MySQL and MariaDB commit DDL
  implicitly, so the statement cannot be rolled back, and an `EXPLAIN` proves only that the grammar
  parsed. A green check mark from a tool that never executed the change is worse than no check.

And whatever the answer is, it has to survive the boring questions: who changed this, when, on which
server, with what proof, and can I put the schema back exactly as it was?

## How we solve it

Five decisions carry the product.

1. **A change is a pair.** A proposal is a statement and its rollback. The kind (DDL or DML) is derived
   from both, and a schema change undone by a data change is refused when it is proposed, not when it
   fails to undo anything.
2. **Rehearsal, not dry run.** The engine's own `SHOW CREATE TABLE` rebuilds the touched tables in a
   throwaway database, the change runs there, the rollback runs there, and the schema must come back
   byte-identical. What the reviewer reads is that diff — plus a row diff for DML. When the server
   will not grant `CREATE DATABASE`, the rehearsal degrades to a *static* check (parse, plan, row
   estimate) and says so by name; it never claims a rollback it did not run.
3. **Claim before statement.** A proposal carries what the rehearsal proved, and the tier is recorded
   as the one that *ran*, not the one that was requested.
4. **Separation of duties.** `read`, `propose`, `approve` and `apply` are separate scopes. The token an
   agent holds proposes; a human's token approves and applies. An override exists and is journaled,
   because a real product is used at 3am.
5. **Artifacts are permanent.** A proposal is never deleted, a target is archived and never removed,
   and the journal is append-only. The state file is one SQLite file with a versioned schema: a
   mismatched version refuses to open rather than silently dropping history.

```
agent ──read──────────────▶ rows
      └─propose──▶ proposal ─▶ rehearsal ─▶ decision ─▶ apply ─▶ journal
                                  │                     │
                     schema diff · row diff       claim before statement
                     locks · algorithm            receipt kept forever
```

## Architecture

One binary, three faces. `http serve` is the API and the only writer of the state file; `web serve`
is the console; every other word is a CLI that speaks to the API and touches no database.

```
                  ┌────────────────────────── one binary ──────────────────────────┐
  agent / CI ───▶ │ cmd/noxis/http      REST API over /v1        (writes state)    │
  browser   ───▶ │ cmd/noxis           CLI: query · propose · approve · apply      │
                  │ cmd/noxis/web       the embedded console (React, orange/white)  │
                  └───────────────┬────────────────────────────────────────────────┘
                                  │  handlers → one use case → one Run
                  ┌───────────────▼────────────────┐
                  │ internal/app                   │  use cases: read, propose,
                  │  (domain + ports only)         │  rehearse, approve, apply, probe
                  └───────┬───────────────┬────────┘
                          │ ports         │ ports
        ┌─────────────────▼──────┐   ┌────▼─────────────────────────┐
        │ internal/adapters/sqlite│  │ internal/adapters/mysql      │
        │ the state file (journal,│  │ one module, two drivers:     │
        │ targets, proposals)     │  │ mysql · mariadb              │
        └─────────────────────────┘  └────────────┬─────────────────┘
                                                  │ MySQL 8 / MariaDB 11
                                   ┌──────────────▼───────────────┐
                                   │ your server                  │
                                   │  app        ← governed        │
                                   │  noxis_rehearse_<rand> ← gone │
                                   └──────────────────────────────┘
```

The dependency direction is enforced by a test, not by review: `cmd/* → app → ports → domain`, and
`adapters/* → ports → domain`. Nothing outside `cmd/` imports `cmd/`, no use case calls another, and
no package outside `internal/common/config` reads an environment variable.

### What lives where

| Path | Role |
| --- | --- |
| `cmd/noxis/` | the binary: `main`, `help`, `wiring` (the composition root), `web` |
| `cmd/noxis/commands/` | the CLI face, one file per command group |
| `cmd/noxis/http/` | the API face: `routes` (the only authority on paths and scopes), `handlers`, `middlewares`, `respond` |
| `cmd/spec/` | post-processes the generated OpenAPI: `x-scopes` from the route table, payload schemas from the annotations |
| `internal/domain/` | entities, the state machine, sentinel errors — stdlib only |
| `internal/ports/` | consumer-owned interfaces |
| `internal/app/` | use cases, one file = one `Run` |
| `internal/adapters/sqlite/` | the state file |
| `internal/adapters/mysql/` | the engine: read, probe, plan, rehearse, apply |
| `internal/client/` | the typed REST client the CLI and the UAT steps both use |
| `web/` | the console sources; `make web` builds it into the binary |
| `test/` | `arch` (layering), `uat` (godog over a real process), `coverage` (floors) |

## Supported platforms

| | |
| --- | --- |
| **Databases** | **MySQL 8.0+** and **MariaDB 10.6+** (tested against 8.4 and 11.4). Nothing else, on purpose: two engines, each one honest. |
| **Host OS** | any Linux that runs a static Go binary; macOS for development. The container is **distroless, non-root (65532), read-only root filesystem**. |
| **Deployment** | Docker / Compose, a systemd unit, or the bare binary. The console runs in the same binary or on another host — the API is the boundary. |
| **State** | one SQLite file. Back it up like a database, because it is the journal. |
| **Telemetry** | OpenTelemetry (OTLP/HTTP) if `OTEL_EXPORTER_OTLP_ENDPOINT` is set; with no endpoint, nothing is exported. |
| **Client** | the CLI is the same binary; the API is plain HTTP + JSON with a generated OpenAPI 3.1 document. |

## Quick start

```bash
docker compose up -d --build          # api :9000 · console :3000 · mysql :3307
open http://localhost:3000            # paste the ops token below
```

The stack ships development tokens, a sealing key, and a MySQL seeded with `users` and `orders`.

| Field | Value |
| --- | --- |
| host / port | `mysql` (the service name) / `3306` |
| schema | `app` |
| username / password | `noxis` / `noxis` |
| TLS | `skip-verify` (the container's certificate is self-signed) |

```
ops    dev_ops_0123456789abcdef0123456789aa    read propose approve apply admin
agent  dev_agent_0123456789abcdef0123456789    read propose
```

The console's **Test connection** button fills the schema from the server, so the first registration
does not need you to remember a database name.

Then:

```bash
make build
./bin/noxis auth signin dev_ops_0123456789abcdef0123456789aa --api-url http://localhost:9000
./bin/noxis target ls
./bin/noxis query <ref> --sql "SELECT * FROM users"
```

## Self-hosting

### 1. The environment

Everything is environment-declared; there are no flags for production settings and no config file to
fall out of sync. `.env.example` is the single list.

| Variable | Meaning |
| --- | --- |
| `TOKENS` | inline JSON: `{"<name>": {"secret": "...", "scopes": [...]}}`. The name is the identity that lands in the journal — name it after the caller, not the permission. Secrets are ≥ 32 characters. |
| `TOKENS_FILE` | the same JSON from a mounted file; wins over `TOKENS`. Use this when escaping JSON in compose is worse than mounting a secret. |
| `SECRET_KEY` | 32 bytes, hex or base64. AES-256-GCM seals target passwords, AAD = the target ref. Unset means a target with a password cannot be created. |
| `QUERY_TIMEOUT` / `PROBE_TIMEOUT` | defaults `30s` / `5s`. |
| `CONSOLE_ORIGINS` | browser origins allowed to call the API, comma-separated. |
| `CONSOLE_HOST` / `CONSOLE_TOKEN` | what the console tells the browser to call, and the token it hands it. The token is proved against `/v1/whoami` at boot: one the server refuses stops the console from listening, instead of serving a sign-in page that reads as "up". |
| `OTEL_EXPORTER_OTLP_ENDPOINT` / `OTEL_SERVICE_NAME` | telemetry, optional. |

Generate the key once and keep it: a row sealed with another key refuses to open and says to re-enter
the password. There is no previous-key support, deliberately — a silent "could not decrypt" is how a
credential leak gets lost in a log.

```bash
openssl rand -hex 32
```

### 2. The container

```bash
docker run -d --name noxis --read-only -p 9000:9000 \
  -e TOKENS='{"agent":{"secret":"…","scopes":["read","propose"]},"ops":{"secret":"…","scopes":["read","propose","approve","apply","admin"]}}' \
  -e SECRET_KEY=<64 hex> \
  -e CONSOLE_ORIGINS=https://console.example.com \
  -v noxisdata:/data noxis:latest
```

- The state file lives at `/data/noxis.db`; the image owns `/data` as uid **65532**. A bind mount must
  be writable by that uid.
- The image is its own healthcheck (`/noxis http healthcheck`): no shell, no curl.
- Behind a reverse proxy, terminate TLS there and keep `CONSOLE_ORIGINS` exact — the API answers CORS
  from that list and nowhere else.

### 3. The console, on its own host

```bash
# VPS A — the API
CONSOLE_ORIGINS=https://console.example.com noxis http serve

# VPS B — the console, told where the API is (there is no baked-in default)
CONSOLE_HOST=https://api.example.com CONSOLE_TOKEN=… noxis web serve --port 3000
```

The console answers `/config.json` with the API address and token it was started with, so one build
serves any deployment.

### 4. What to back up

`/data/noxis.db` — targets, proposals, rehearsals and the append-only journal. Treat it as a database:
snapshot it, restore it, never edit it. The schema is versioned with `PRAGMA user_version`; a version
mismatch refuses to open with a reset instruction rather than migrating silently.

### 5. Grants on the governed server

A rehearsal creates and drops a throwaway database:

```sql
GRANT CREATE, DROP ON *.* TO 'noxis'@'%';
GRANT ALL PRIVILEGES ON `noxis_rehearse_%`.* TO 'noxis'@'%';
```

Without them nothing breaks: the rehearsal degrades to a static check and its note names the engine
error and the grant that would restore the real one.

## Let an agent use it

noXis is built for this case, so the setup is explicit rather than implied.

### First install: ask, do not assume

An agent installing noXis must not pick its own capabilities. Before it writes a token, it asks the
operator four questions and waits for the answers:

1. **Which server, and which schema?** Host, port, engine, and the one schema this target governs.
   (MySQL and MariaDB call a schema a database; a server with several becomes several targets.)
2. **What may I do, and what stays yours?** The five capabilities below are separable — an agent that
   reads and proposes never needs `approve` or `apply`.
3. **Who decides?** Approving and applying should be a different token holder. Say who, and say
   whether an override may exist.
4. **May I register and archive instances?** That is `admin`, and it is not needed to change a schema.

Then write one token per caller and keep the names honest:

```bash
TOKENS='{
  "agent-claude": {"secret": "…", "scopes": ["read", "propose"]},
  "ops-dimas":    {"secret": "…", "scopes": ["read", "propose", "approve", "apply", "admin"]}
}'
```

The name — not the token — is what the journal records, so an incident review answers "who" in words.

### What the agent can do, and with which scope

| Capability | Scope | Command |
| --- | --- | --- |
| See what this token may call, and the contract | any token | `noxis describe`, `noxis whoami`, `curl :9000/openapi.json` |
| List the instances it governs | `read` | `noxis target ls`, `noxis target show <ref>` |
| Find what is in a server | `read` | `noxis databases <ref>`, `noxis tables <ref> app`, `noxis table <ref> app users` |
| Query rows | `read` | `noxis query <ref> --sql "SELECT * FROM users LIMIT 10"` |
| Propose a DML or DDL change | `propose` | `noxis propose <ref> --title "…" --sql "…" --rollback "…"` |
| Re-run the proof | `propose` | `noxis rehearse <ref>` |
| Look up changes | `read` | `noxis proposals --target <ref>`, `noxis show-proposal <pr>`, `noxis events <pr>` |
| Wait for a decision | `read` | `noxis watch <pr> --until applied,failed` |
| Approve or reject | `approve` | `noxis approve <pr> --note "…"` / `noxis reject <pr>` |
| Apply an approved change | `apply` | `noxis apply <pr> --wait` |
| Correct a connection | `admin` | `noxis instance set <ref> --password … --probe` |
| Register or archive an instance | `admin` | `noxis instance add …`, `noxis instance archive <ref>` |

`noxis describe` is the agent's first call in a new environment: it reads the running server's
OpenAPI, joins each operation with `x-scopes`, and answers a table (or JSON) of what this caller may
actually do. An agent that starts there cannot promise a capability it does not hold.

### The loop an agent should follow

```
describe ─▶ whoami ─▶ target ls ─▶ query (read the truth) ─▶ propose ─▶ watch ─▶ report
                                                    │
                                    rollback is mandatory for DDL:
                                    the rehearsal runs it and diffs the schema back
```

Every command speaks JSON with `--json`, and every failure is `{"error":{"code","message","status"}}`
on stderr with a non-zero exit code. `code: "client"` means the request never left the machine — that
is the agent's bug, not the server's.

An MCP adapter is a thin layer over `internal/client` (six tools, two resources) rather than a wrapper
around the binary; it is on the roadmap, not a promise.

## The API

- **Refs, never integer ids.** Every row is keyed by a UUIDv7 `ref` — the only identifier on the wire.
- **Cursors, never offsets.** Lists take `?limit=` and `?cursor=<ref>`; `meta.next_cursor` is absent on
  the last page. Raw `SELECT` results are the exception: they report `limit` + `truncated`, because a
  keyset cannot be inferred from arbitrary SQL and a fake cursor would lie.
- **One envelope.** `{"data": …, "meta": …}` on success, `{"error": {"code","message"}}` on failure.
- **Scopes in the contract.** The OpenAPI document names each operation's payload type under `data` and
  carries `x-scopes`, both copied from the code that serves it. `make spec` regenerates it and CI fails
  on drift.
- **Idempotency.** Both creates accept `Idempotency-Key`; a replay answers `200` with the row it made.

## The console

`noxis web serve` serves the console the binary embeds — React 19, Ant Design, Monaco (bundled, never a
CDN), orange on white.

- **Instances** — one row per governed server, its engine, version and last probe.
- **Instance** — the facts, connectivity, an **Edit** drawer that proves new credentials before saving,
  and three tabs: **Query** (a draft kept per instance, results, CSV), **Tables** (the schema and its
  tables beside the rows), **Changes** (the queue, with a proposal drawer).
- **Proposal** — verdict band, the change and its rollback, the rehearsal evidence as a real diff, and
  the decision history.

Keyboard: rows are links (`Enter`/`Space`), drawers close on `Esc` and return focus, and every control
has a visible focus ring. No horizontal scrolling at 360px.

## Releases and updates

Every green commit on `master` is published. The publish job runs after `go` and `web` pass, builds the
binary for `linux/amd64` and `linux/arm64` with the console embedded, and moves one moving release —
the tag **`nightly`** — to that commit:

| Asset | What it is |
| --- | --- |
| `noxis-linux-amd64`, `noxis-linux-arm64` | the binary, stamped `Version=nightly`, `Commit=<first 7 of the commit>` |
| `version.json` | `{"version":"nightly","commit":"db92d5a","built_at":"…"}` — the artifact clients read |
| `SHA256SUMS` | the checksums of both binaries |

The version a build reports is the release line plus the seven characters of its commit:
`nightly+db92d5a`. `noxis version` prints it, the console shows it in the topbar, and `/healthz` and
`/v1/version` answer with it. A build nobody stamped says `dev` and makes no claim.

**The update check runs on boot.** The CLI asks its server after every command and the console asks on
every page load; the *server* is what talks to GitHub, at most once an hour, and remembers the answer.

```
client ── GET /v1/version ──▶ this deployment: {version, commit, release_url, latest, newer}
                                     │
                                     └─ checks (1h cache) ──▶ …/releases/download/nightly/version.json
```

A newer build is reported as a line on stderr (`a newer build is out: nightly+abc1234`) and a
dismissible banner in the console. Nothing installs itself: every notice links to
**https://github.com/dimasbaguspm/noxis/releases**, and updating is a download and a restart. A
deployment with no outbound network says `latest: null` and behaves exactly as before.

### Cutting a release

1. Everything intended for it is merged; CI is green on `master`.
2. Tag it: `git tag -a v2.0.0 -m "v2.0.0" && git push origin v2.0.0`.
3. Build the release build with the tag as the release line:
   `make build VERSION=v2.0.0` (the commit is stamped automatically).
4. Create the release from that tag, attach the same three kinds of asset as the nightly, and publish
   it (not as a pre-release). The nightly keeps moving; a tagged release never does.
5. The next `/v1/version` a client reads names the newer commit, so every deployment with an admin
   watching the console learns it exists.

The image is built per deployment (`docker build -t noxis .`) and carries the same stamp:
`--build-arg VERSION=v2.0.0 --build-arg COMMIT=$(git rev-parse --short=7 HEAD)`.

## Contributing

Read `AGENTS.md` first: it is the map, and `docs/agent/` is the detail.

1. **Epic → tickets → one pull request per ticket.** A ticket is a vertical slice that cuts every layer
   and can be demonstrated alone. Drafts are shown before issues are opened.
2. **A branch per ticket.** Title `[ai:y][#<issue>] <type>(<scope>): <subject>`; the body closes the
   issue and says what was verified, not what was intended.
3. **CI must be green** before review: `go` (lint, mocks drift, spec drift, build, test, uat, coverage
   floors) and `web` (install, typecheck, lint, format, unit tests, build).
4. **Review against the ticket's "Done when".** The human merges; nothing lands on `master` directly.

```bash
make test          # unit + the layering test                     (~5s)
make lint          # golangci-lint v2                            (~30s)
make uat           # godog over a real `noxis http serve` process (~20s)
make cover         # per-package coverage against test/coverage/floors.txt
make uat-slow      # the same plus @slow: boots MySQL 8.4 + MariaDB 11.4 (docker)
make spec          # regenerate the OpenAPI document (committed)
make mocks         # regenerate the generated mocks (committed)
make web           # build the console into the binary
```

Conventions that are not negotiable: the domain stays stdlib-only, a use case never calls another, no
environment read outside `internal/common/config`, no hardcoded scope or driver or status, errors are
sentinels wrapped with `%w`, and a `nolint` names the lint and the reason. Coverage floors are a
ratchet — they only go up.

## Verify

```bash
make test && make lint && make uat
pnpm -C web check && pnpm -C web test
```

## Roadmap

- **MCP adapter** over `internal/client`: the CLI is the agent's primary interface today.
- **Multi-schema targets** — today one target governs one schema; a server with many becomes many
  targets, which keeps the review boundary where it belongs.
- **Row diff for DDL**, chunked applies for large tables, and a policy layer for "this DDL is refused
  outright".

## FAQ

**Why only MySQL and MariaDB?** Because a rehearsal is engine-specific: the throwaway rebuild, the
implicit DDL commit, the lock algorithm and the row-image format differ per engine. Two engines done
honestly beat thirteen done by analogy.

**Why not just a dry run?** MySQL and MariaDB commit DDL immediately; there is nothing to roll back.
A rehearsal that never executed the change proves nothing, which is why the tier is recorded as the one
that ran — and why `static` never claims the rollback was proven.

**Does it need a sidecar?** No. One binary, one SQLite file, one HTTP API.

**What if the token leaks?** It can only do what its scopes allow, every action is journaled under the
token's name, and the proposal gate means a leaked `propose` token cannot apply anything.

## Status

v2. The v1 tree — thirteen engines, dry-run verifier — is frozen on the `v1` branch and the `v1.0.0`
tag, and is the reference for anything this rebuild has not reached.

## License

Not declared yet. A `LICENSE` file is the one thing this repository still needs before a stranger can
safely self-host it; until then, treat it as source-available and ask.
