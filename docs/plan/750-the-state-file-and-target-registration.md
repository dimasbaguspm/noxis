# [api]: the state file and target registration

Plan issue #750, PR #755, merged 2026-11-20.

## What problem does this solve?

The API could authenticate a caller and answer `/whoami`, and that was all. Before an agent can read or propose, the server needs the one file that remembers which databases it governs, who registered them and what happened to them.

## What is the user facing?

`POST /v1/targets` registers a MySQL or MariaDB server with an operator token and answers a `ref` — never a password. `GET /v1/targets` lists newest-first with `?limit=` and `?cursor=`, `GET /v1/targets/{ref}` reads one, `PATCH` renames or rotates a password, `DELETE` archives. `noxis target add | ls | show | archive` do the same from the CLI.

## How will I solve it?

`internal/adapters/sqlite` opens the file with WAL, `busy_timeout`, a single writer and `STRICT` tables, and gates the schema on `PRAGMA user_version`: a foreign file or an unknown version is refused by name, never migrated. Two tables keyed by a UUIDv7 `ref`: `targets` and `journal`. `domain.Target` validates itself; `ports` declares `Targets`, `Journal` and `Sealer`; `internal/app/target` holds one file per use case with one `Run`; `cmd/http` carries the five routes with their scopes; `internal/client` and `cmd/cli` carry the commands. Passwords are sealed with AES-256-GCM, AAD = the target `ref`, and the wire type has no password field at all.

## Done when

- [x] `make test` covers the schema-version guard, foreign-file refusal, ref-only lookups, cursor paging and the archive rule
- [x] A password appears in no response body and no log line; an unset `SECRET_KEY` refuses the write
- [x] `@fast` UAT proves create → list → get → archive, 403 without `admin`, 400 for a non-reference
- [x] Archiving an unknown target is 404 and archiving twice is 409

## Risks & notes

- `proposals` and `rehearsals` are deliberately absent; they arrive with their own tickets and bump `user_version`.
- Archive is the only removal, and the name stays unique among archived rows, so re-registering the same server means renaming.
- The operator token needs `admin` to register; the agent token holds only `read` and `propose`, which is the separation this ticket depends on.
