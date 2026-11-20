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
| `noxis web serve` | the console, served from the same binary (see #789) |
| `noxis <command>` | a client of that API — `auth`, `target`, `query`, `propose`, `review` |
| `web/*` | a standalone console, later; not embedded, talks to the same API |

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
