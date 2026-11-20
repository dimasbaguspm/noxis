# [api]: approve and apply a proposal, and keep the receipt

Plan issue #754, branch `feat/754-decide`.

## What problem does this solve?

A rehearsed proposal is still only evidence. Someone has to decide, the decision has to be attributable, the proposer has to be refused unless an admin overrides it in the open, and the statement has to run exactly once with a receipt that cannot be edited afterwards.

## What is the user facing?

`POST /v1/proposals/{ref}/approve` and `/reject` take a note and record who decided. `POST /v1/proposals/{ref}/apply` runs the approved statement and records who applied it and what the engine said. An irreversible change needs `{"confirm_destructive": true}`. `GET /v1/proposals/{ref}/events` is the whole history. `noxis approve | reject | apply | events` do the same from the CLI.

## How will I solve it?

The state machine in `internal/domain` already refuses illegal transitions; this ticket makes it reachable and adds the rules on top: an approval needs `status=verified`, the proposer's own token is refused unless it holds `admin` (and that exception is its own journal line, kind `override`), and a rejection may be the proposer withdrawing. `apply` claims the proposal with a conditional `UPDATE … WHERE status='approved'` **before** the statement runs, so a crash leaves `applying` — indeterminate and unmistakable — and two operators cannot both apply. `Driver.Apply` wraps a DML statement in one transaction; a DDL statement cannot be wrapped, because both engines commit it implicitly, and pretending otherwise would print a receipt for a rollback that never happened. The event list reads the append-only journal.

## Done when

- [x] The proposer's token is refused with 403 on approve, and the `admin` override is a journal row
- [x] Two concurrent applies of one proposal run the statement once (the claim decides)
- [x] A failed apply stores the failure and the proposal stays `failed`
- [x] Approving a proposal whose rehearsal failed is a 409 with the reason
- [x] An irreversible change needs `confirm_destructive`, and a 400 says so
- [x] `@slow` UAT proves propose → verify → approve → apply → the change is in the engine, with the events list showing `approved` and `applied`

## Risks & notes

- `TRUNCATE`/`DROP` with no rollback rehearses as `partial`. That is now `Usable`: without it a destructive change could never be approved, which is the same as pretending nobody ever runs one. The `irreversible` flag plus the typed confirmation is the honest gate, not an unreachable state.
- Apply is not idempotent and cannot be made so; the claim plus the receipt is the only defence.
- `applying` after a crash is never retried automatically: the operator looks at the target first.
