# [api]: rehearse a proposal — shadow rebuild with schema and row diffs

Plan issue #753, branch `feat/753-rehearsal`.

## What problem does this solve?

MySQL and MariaDB commit DDL implicitly, so a transaction proves nothing for a schema change. Without somewhere else to run it, a reviewer sees a statement and a promise. This ticket makes the promise into evidence: the change is executed in a throwaway database rebuilt from the engine's own DDL, and the rollback has to put that database back exactly as it was.

## What is the user facing?

`POST /v1/targets/{ref}/proposals` takes `{title, query, rollback, note, tier}` and answers the proposal with the rehearsal that judged it: status, tier, schema diff, row diff, `irreversible`, and a note saying how far the claim goes. `GET /v1/proposals` lists the queue newest-first, `GET /v1/proposals/{ref}` reads one with its evidence, `POST /v1/proposals/{ref}/rehearse` re-runs the proof. `noxis propose`, `noxis proposals`, `noxis show-proposal` and `noxis rehearse` do the same from the CLI.

## How will I solve it?

`internal/adapters/mysql/rehearse.go` creates `noxis_rehearse_<random>`, mirrors every table the change and its rollback name from `SHOW CREATE TABLE`, runs the change rewritten to that database, snapshots every table's DDL, runs the rollback rewritten the same way, and requires the snapshot to come back byte-identical (the `AUTO_INCREMENT` counter is normalised away). Three tiers, and the proposal records which one it got: `static` (parse, `EXPLAIN`, `information_schema` estimate — never better than `partial`), `shadow` (the rebuild above), `rehearsal` (the same plus up to 200 sampled rows, so row counts mean something). A statement naming another schema is refused rather than rewritten. `domain.Proposal` owns the state machine and refuses illegal transitions; every attempt is stored, failures included.

## Done when

- [x] A DDL proposal against real MySQL and MariaDB comes back with a schema diff and a proven reversal, and the scratch database is gone afterwards
- [x] A DML proposal reports the rows it touched (`tier=rehearsal`) and proves the rollback restored them
- [x] A rollback that does not restore the schema fails the proposal, with the diff showing why
- [x] A statement with no rollback is marked `irreversible` and the proposal is stored as `failed`
- [x] Without `CREATE DATABASE` the rehearsal degrades to `unsupported` and says so by name
- [x] `@slow` UAT covers both engines

## Risks & notes

- `CREATE INDEX … ON t` is not in the lexer's scope list yet, so its rehearsal runs against an empty throwaway database and fails visibly. Visible is the right failure; the scope list is the fix.
- The sampled rows of `tier=rehearsal` are a sample: the note says so, and the plan is never to claim the whole table.
- Triggers are not replayed into the throwaway database. A change touching a table with triggers is rehearsed without them, and the note does not yet say that — the next ticket should.
- `approved`/`rejected`/`applied` exist in the state machine and the schema but no route reaches them: that is #754.
