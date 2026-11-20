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
| `noxisd` (`cmd/http`) | the server: REST API over `/v1`, the only writer of the state file |
| `noxis` (`cmd/cli`) | a client of that API — `whoami`, and later `target`, `query`, `propose`, `review` |
| `web/*` | a standalone console, later; not embedded, talks to the same API |

## Run it

```bash
cp .env.example .env            # set TOKENS, SECRET_KEY
make run                        # noxisd on :9000
curl -s localhost:9000/healthz
curl -s -H "Authorization: Bearer <secret>" localhost:9000/v1/whoami
```

```bash
make build && API_TOKEN=<secret> ./bin/noxis whoami
```

Supported engines: **MySQL 8** and **MariaDB 11**, both through one module registered
under two driver names.

## Verify

```bash
make test    # unit + the layering test that keeps domain pure
make lint
make uat     # godog, against a real noxisd process
```

## Status

v2, in progress. The v1 tree — thirteen engines, embedded console, dry-run verifier —
is frozen on the `v1` branch and the `v1.0.0` tag, and is the reference implementation
for anything this rebuild has not reached yet.
