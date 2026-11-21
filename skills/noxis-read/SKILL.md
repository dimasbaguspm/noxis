---
name: noxis-read
description: Read a governed MySQL or MariaDB instance through noXis — list instances, schemas and tables, run SELECT/EXPLAIN/SHOW, and look up existing proposals. Use whenever a question about the data can be answered without changing it.
---

# noXis read

Reading is boring on purpose: `read` scope gets rows immediately, and the read route refuses to become a
side door for a write. Only `SELECT`, `EXPLAIN` and `SHOW` run.

## Start here

```bash
noxis api describe                                 # the API this server serves, and which of it your token may call
noxis auth whoami                                   # name and scopes
noxis target ls --json                         # the instances you govern
```

`describe` is the first call in a new environment: it joins the running server's OpenAPI with your scopes,
so you cannot promise a capability you do not hold.

## Find the data before you query it

```bash
noxis target show <ref>                        # engine, version, schema, last probe
noxis table <ref>                              # every schema those credentials can see
noxis table <ref> app                          # names, engine, estimated rows
noxis table <ref> app users                    # columns, indexes, and the DDL
```

Two things worth knowing:

- A target governs **one** schema. Other schemas are readable; only the governed one can be changed.
- `tables` reports an estimate from `information_schema`, not a count. A count costs what a reviewer is
  trying to avoid.

Every `<ref>` here is forgiving: the **name** `target ls` prints, a unique **prefix** of the reference, or
the whole thing. An ambiguous prefix is refused and lists what it matched — nothing is guessed.

## Query

```bash
noxis query <ref> --sql "SELECT id, email FROM users LIMIT 20" --json
noxis query <ref> --sql "EXPLAIN FORMAT=JSON SELECT * FROM users WHERE email = 'x'" --json
noxis query <ref> --file report.sql --limit 500 --json
```

- `--limit` defaults to 200, maximum 1000. The answer carries `row_count` and `truncated`: when
  `truncated` is true the result was cut by the ceiling, not by the data.
- There is no cursor for raw SQL, and no fake one: a keyset cannot be inferred from arbitrary SQL.
- Write `SELECT` that can run on a replica: no `FOR UPDATE`, no long transactions.
- The statement arrives as one argument. Prefer `--file` for anything over a line.

## Look up changes that already exist

```bash
noxis proposal ls --target <ref> --json
noxis proposal ls --status verified --limit 50 --json
noxis proposal show <pr> --json                # the change, its rollback, and the rehearsal
noxis proposal events <pr> --json              # the append-only history
noxis proposal watch <pr> --until applied,failed --timeout 5m
```

Check `proposals` before proposing: a change that is already queued or applied is a duplicate, and the
journal is the answer to "was this already tried?".

## Failure

| Code | Means | Do |
| --- | --- | --- |
| `client` | the request never left the machine | fix the command or the profile; do not retry blindly |
| `unauthorized` | the token is not accepted | `noxis auth status`, then ask the operator |
| `forbidden` | the scope is missing | say which scope you need; do not ask for another token silently |
| `unsupported` | the statement is not a read | this is a change — use `noxis-propose` |

## Never

- Never modify data through `query`. If it does not start with `SELECT`, `EXPLAIN` or `SHOW`, it is not a
  read; the server will refuse it, and trying is how a `read` token becomes a write token.
- Never print a full result set into a report: quote the columns, the row count, and the query you ran.
