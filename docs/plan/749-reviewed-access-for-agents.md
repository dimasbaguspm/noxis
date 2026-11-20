# v2 epic: reviewed access to MySQL and MariaDB for agents

Plan issue #749, closed 2026-11-20. Five tickets, six pull requests.

## The bet

An agent reads through `read`, proposes DDL/DML through `propose`, and a human holding `approve`+`apply`
decides. The proposal arrives rehearsed — the schema rebuilt in a throwaway database, the change run
there, the rollback required to put it back — so the reviewer reads evidence rather than a check mark.

## What shipped

| Ticket | What it added | PR |
| --- | --- | --- |
| #750 | the state file (`PRAGMA user_version`, `STRICT`, one writer), targets, sealed passwords, journal | #755 |
| #751 | one engine adapter for MySQL and MariaDB, the probe, capabilities derived from the engine | #756 |
| #752 | the read guard (`sqltext`), bounded reads with `truncated`, schema metadata | #757 |
| #751/#752 gap | the `@slow` harness: containers booted, waited on, removed by the suite | #758 |
| #753 | proposals + rehearsals, the shadow rebuild, schema and row diffs, the tier ladder | #759 |
| #754 | approve, reject, apply — separation of duties, the claim, the receipt, the events list | #760 |

## Verified, not asserted

- `make test` — unit tests for every package plus 22 `@fast` scenarios against a real `noxisd`.
- `make lint` — 0 issues.
- `UAT_SLOW=1 make uat-slow` — the same plus 9 `@slow` scenarios against `mysql:8.4` and
  `mariadb:11.4`: a live probe, a live read, a refused write against a reachable engine, a verified
  DDL diff, a lying rollback failing with its diff, a sampled DML run, a destructive change marked
  irreversible, and the full propose → approve → apply path landing a column in the engine.
- `test/arch` — the dependency direction is a test, not a convention.

## Decisions worth remembering

- **Rehearsal, not dry run.** Both engines commit DDL implicitly, so a transaction proves nothing for
  a schema change; the proof is a rebuilt database and a diff.
- **`partial` is approvable.** A change with no rollback rehearses as `partial`; refusing to approve
  it would make every destructive statement unreachable. `irreversible` plus `confirm_destructive`
  is the gate, and the note says exactly what was not proven.
- **The claim comes before the statement.** Apply moves `approved → applying` with a conditional
  update first, so a crash is indeterminate and visible rather than silent, and two operators cannot
  both apply.
- **Identity is the token's name.** No user table; the journal records the name that acted, and an
  admin overriding separation of duties is its own journal line.
- **Clients see `ref`, never an integer id.** UUIDv7, so `ORDER BY ref DESC` is newest-first and a
  keyset at the same time.

## What is not done

- Triggers are not replayed into the rehearsal database, and the rehearsal note does not say so.
- `CREATE INDEX … ON t` is not in the lexer's scope list; its rehearsal fails visibly against an
  empty throwaway database.
- A proposal stuck in `applying` after a crash has no route to inspect or clear it.
- `web/` — the standalone console — has not been started. The API it will consume is complete for
  the flow above.
- The v1 issues #744–#747 describe a module contract that v2 replaced; they are closed with a
  pointer here.
