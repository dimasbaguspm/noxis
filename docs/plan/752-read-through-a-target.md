# [api]: read through a target — guarded query and schema metadata

Plan issue #752, branch `feat/752-read`.

## What problem does this solve?

The product bet starts with an agent that can read. Until a `read` token can run a SELECT and get a bounded, honest answer — and see the schema it is writing against — there is nothing for a proposal to be based on.

## What is the user facing?

`POST /v1/targets/{ref}/query` takes `{"sql", "limit"}` and answers columns, rows, `row_count`, `truncated` and `duration_ms`. Only SELECT, EXPLAIN and SHOW pass; anything else is a 400 naming the kind it refused. `GET …/databases`, `…/databases/{db}/tables` and `…/databases/{db}/tables/{table}` answer the schema — columns, indexes, engines, row and size estimates, and the `SHOW CREATE TABLE` text a rehearsal will rebuild from. `noxis query | databases | tables | table` print them.

## How will I solve it?

`internal/common/sqltext` classifies a statement and names its tables: comments and string literals are blanked first, so a keyword inside a literal cannot change the verdict, and a second statement in one request is refused rather than read past. `internal/app/read` holds four use cases — query, databases, tables, table — each applying `QUERY_TIMEOUT` and the row ceiling. `internal/adapters/mysql` runs them: it reads at most `limit+1` rows to know whether it truncated, keeps NULL as NULL, quotes every identifier from a path, and maps the engine's error numbers onto `ErrNotFound`, `ErrInvalid`, `ErrForbidden` or `ErrUnsupported` with the password scrubbed from any message.

## Done when

- [x] A statement that is not SELECT, EXPLAIN or SHOW is refused with a 400 naming it, including `WITH … INSERT`, a multi-statement, and `SELECT … FOR UPDATE`
- [x] A result larger than the ceiling reports `truncated: true` and no fake cursor
- [x] Metadata answers for both engines, and a missing table is a 404 carrying the engine's message
- [ ] `@slow` UAT proves the guard and the truncation against real containers — verified by hand this round, the harness is still `@fast`

## Risks & notes

- The guard runs before any connection, so a write is refused even when the target is reachable; the `@fast` scenarios use a closed port to prove exactly that.
- Column types are reported as the engine names them, not normalised: one string per engine beats a lie that they are the same.
- Metadata lists are bounded by the schema, not by a cursor: they are the engine's names, not our rows, so there is no keyset to follow.
- Reads are not journalled. A read per audit line would turn an audit trail into a traffic log; the journal is for changes.
