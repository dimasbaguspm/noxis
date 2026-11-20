# noXis

**Reviewed access to the MySQL and MariaDB you already run.**

An agent (or a person) reads your databases through one API. When something needs to
change, it writes the statement and the statement that undoes it, and gets back a
rehearsal: the schema diff, the rows it touches, the locks it takes. A human approves
what the rehearsal showed. Then it runs.

Nothing is installed beside your database. One binary, one state file, one API.

## The two paths

**Reading is boring on purpose.** `read` scope gets `SELECT`, `EXPLAIN` and `SHOW`
back immediately, with paging and a row ceiling. The read route refuses to become a
side door for a write.

**Writing is a proposal, not a keystroke.** `propose` scope creates a proposal: the
statement, its rollback, and a note. The proposal is rehearsed in a throwaway
database rebuilt from `SHOW CREATE TABLE`, so the reviewer sees what changes and what
comes back, not a green check mark. `approve` and `apply` are separate scopes, held by
a different token than the one that proposed.

```
agent ──read──────────────▶ rows
      └─propose──▶ proposal ─▶ rehearsal ─▶ review ─▶ apply ─▶ journal
                                  │                     │
                       schema diff · row diff     one transaction
                       locks · algorithm          receipt kept forever
```

## Runtimes

| Binary | What it is |
| --- | --- |
| `noxis http serve` | the server: REST API over `/v1`, the only writer of the state file |
| `noxis web serve --api <url>` | the console, served from the same binary on `:3000` (`--port`, `--api`) |
| `noxis <command>` | a client of that API — `auth`, `target`, `query`, `propose`, `review` |
| `web/*` | a standalone console, later; not embedded, talks to the same API |

## Try it locally with compose

```bash
docker compose up -d --build      # api :9000, console :3000, mysql :3307
open http://localhost:3000        # paste the ops token below
```

The stack comes with development tokens, a sealed-password key, and a MySQL seeded with `users` and
`orders` so a query returns something. Register it from the console (or the CLI):

| Field | Value |
| --- | --- |
| host / port | `mysql` (the service name) / `3306` |
| database | `app` |
| username / password | `noxis` / `noxis` |
| TLS | `skip-verify` — the container's certificate is self-signed and `verify` would refuse it |

```
ops    dev_ops_0123456789abcdef0123456789aa    read propose approve apply admin
agent  dev_agent_0123456789abcdef0123456789    read propose
```

Every value is overridable in a `.env` beside `docker-compose.yml` (`API_PORT`, `CONSOLE_PORT`,
`MYSQL_PORT`, `TOKENS`, `SECRET_KEY`, `CONSOLE_ORIGINS`, `MYSQL_IMAGE`). MySQL is published on **3307**
so it never fights a MySQL you already run. `docker compose down -v` throws the state file away with
the stack; without `-v` the journal survives a restart.

The same binary serves both faces: `api` runs `http serve` and owns the state file, `console` runs
`web serve --api http://localhost:9000` — the address the *browser* reaches, not the service name.

## The API and the console may live on different hosts

The console is a page in a browser, so it calls the API at the address *the browser* can reach, not
the one the console process happens to see. The API, in turn, has to allow that page's origin.

```bash
# VPS A: the API, and the origin the console will be served from
CONSOLE_ORIGINS=https://console.example.com noxis http serve

# VPS B: the console, told where the API is (required: there is no default)
noxis web serve --api https://api.example.com --port 3000
```

The console answers `/config.json` with `{"api": "https://api.example.com"}`, so a build learns its
API from the deployment instead of carrying one baked into a bundle. Nothing else changes when the
two run on one machine: `--api http://localhost:9000` is the local case.

## Run it

```bash
cp .env.example .env            # set TOKENS, SECRET_KEY
make run                        # the API on :9000
curl -s localhost:9000/healthz
curl -s -H "Authorization: Bearer <secret>" localhost:9000/v1/whoami
```

```bash
make build
./bin/noxis auth signin <secret> --api-url http://localhost:9000   # once; stored at 0600
./bin/noxis target ls
```

`auth signin` proves the token against the server before storing it, `auth status` prints
the server, caller and scopes (never the token), and `auth signout` deletes it. Every command
re-checks `whoami` on boot, cached for a minute; a token the server no longer accepts says
`auth signin` instead of failing halfway through a rehearsal. For CI, `--token` and
`API_TOKEN` still work, and say once that they are the less private path.

Supported engines: **MySQL 8** and **MariaDB 11**, both through one module registered
under two driver names.

The contract is served by the API itself: `curl localhost:9000/openapi.json` (or `.yaml`) describes
every route, its parameters, its bodies and the scopes it needs. A test walks the route table
against that document, so a route nobody documented fails CI.

In a container: one binary, distroless, non-root, read-only root filesystem.

```bash
docker build -t noxis .
docker run -d --name noxis --read-only -p 9000:9000 \
  -e TOKENS='{"agent":{"secret":"...","scopes":["read"]}}' -e SECRET_KEY=<64 hex> \
  -v noxisdata:/data noxis
```

The state file lives in `/data`, which the image owns as `nonroot`; a bind mount instead of a
named volume must be writable by uid 65532. The image is its own healthcheck
(`/noxis http healthcheck`), so it needs no curl and no shell.

## Verify

```bash
make test    # unit + the layering test that keeps domain pure
make lint
make uat     # godog, against a real `noxis http serve` process
```

## Status

v2, in progress. The v1 tree — thirteen engines, embedded console, dry-run verifier —
is frozen on the `v1` branch and the `v1.0.0` tag, and is the reference implementation
for anything this rebuild has not reached yet.
