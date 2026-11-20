# Review checklist

Read the ticket's `Done when` first, then check these. Anything here that fails is a
review blocker, not a style note.

## Layering

- [ ] `internal/domain` imports nothing outside the standard library (`make test` runs `test/arch`).
- [ ] No use case calls another use case.
- [ ] No SQL outside `internal/adapters/sqlite`, no HTTP outside `internal/httpapi`.
- [ ] A new port is described in the caller's vocabulary, not the adapter's.
- [ ] No `NOXIS_` env read outside `internal/common/config`.

## Identity and access

- [ ] Every new route appears in `internal/httpapi/routes.go` with its scopes or `Public: true`.
- [ ] A scope check is never done inside a handler — it comes from the route table.
- [ ] Anything on the wire is a `ref`; no integer id, no `rowid`, no name-keyed filter.
- [ ] A secret, a password, or a cipher appears in no response and no log line.
- [ ] `proposed_by` cannot equal `decided_by` without the `admin` exception, and the
      exception writes a journal row.

## Lists and reads

- [ ] Lists take `limit` + `cursor` and return `meta.next_cursor`; `OFFSET` appears nowhere.
- [ ] A read route cannot be turned into a write (`SELECT`/`EXPLAIN`/`SHOW` only, checked
      by the lexer, not by a prefix match).
- [ ] Raw result sets report `truncated` rather than pretending to page.

## Proposals

- [ ] A proposal that has been approved is immutable except for apply state.
- [ ] `apply` runs in one transaction; a failure leaves `failed` with the cause, never
      a silent retry.
- [ ] A rehearsal result is stored, including `failed` and `unsupported` — the reviewer
      sees what the machine actually did.
- [ ] Anything destructive carries `irreversible` and requires `confirm_destructive`.

## Durability

- [ ] The state file schema changed ⇒ `PRAGMA user_version` bumped, no migration path added.
- [ ] `make cover` passes and a package whose coverage rose has its floor raised in the same PR: a floor is a ratchet, never a ceiling to lower.
- [ ] The journal is append-only; nothing in the codebase deletes from it.
- [ ] Targets archive; they are never hard-deleted.

## Tests

- [ ] A new endpoint has a `@fast` scenario (fake driver, no container).
- [ ] A new SQL behaviour has a `@slow` scenario against real MySQL and MariaDB.
- [ ] The happy path is proven end to end: read → propose → rehearse → approve → apply.
