# [plan]: the agent's seat — what it can learn, wait for, and retry

Plan issue #833, closed 2026-11-21. Four tickets and one fix, each its own branch and pull request.

## What problem does this solve?

The CLI is the agent's interface: an agent signs in, reads, proposes, and asks a human to decide. Four things it could not do made it a bad client. A failure arrived as English prose on stderr, so the only way to learn that a target was not found was to match the sentence — while the API had already answered with a code. A change applied in the background answered 202 and the agent had no way to wait for the verdict except busy-looping `show-proposal` with its own timeout and its own idea of which statuses are terminal. A list stopped at one page, so reading a queue meant implementing keyset paging in the caller. And a retried create made a second row: a timeout cost a duplicate instance, or a duplicate proposal somebody had to reject by hand.

The human side had the matching gap: the proposal page showed everything and said nothing. A reviewer scrolled three cards of SQL and diffs before learning whether the rehearsal had proved the change, and the buttons that decide were in the footer of the first card.

## What is the user facing?

`--json` answers a failure as `{"error":{"code","message","status"}}` on stderr, in the API's own vocabulary, and keeps a non-zero exit. `noxis watch <ref>` waits for a status a proposal does not leave, `apply --wait` attaches that wait to the call that started it, and `--all` walks every page. `noxis describe` answers what this deployment offers and which of it this token may call — a table for a person, one document for an agent. A create may carry an `Idempotency-Key`: the retry answers the row the first call made, with `200` instead of `201`. The proposal page opens with a verdict, one derived sentence plus the facts it rests on, and the decision sits in a sticky aside.

## What shipped

| Ticket | What landed | PR |
| --- | --- | --- |
| #834 | `commands.Report` shapes a failure on stderr as JSON, with the API's code or `code: client` when the request never left; the CLI's own courtesies become JSON lines too. `watch` polls to a settled status with `--until`, `--interval`, `--timeout`; `apply --wait` reuses it. `walkAll` pages every list and stops at `constants.MaxPages`. `target ls --all` (archived) became `--archived` | #838 |
| #835 | `noxis describe` reads `/openapi.json` from the running server, joins each operation's `x-scopes` with `whoami`, and answers `allowed` per operation. `client.OpenAPI` and a `send` that returns the status split out of `do`; `constants.Tiers` with a domain test that keeps the two lists identical | #839 |
| #836 | `Idempotency-Key` on both creates, stored as a nullable unique column on `targets` and `proposals` (**schema v5**); a replay answers the row the first call made with `200`, the unique index settles a race, and the client names every create unless the caller pins one | #840 |
| #837 | `verdictOf` + a verdict band above the page: one derived sentence, bounded by what the tier actually ran, plus the facts it rests on. Two columns above 1024px with the decision in a sticky aside | #841 |
| #833 | `watch` was built, tested and never registered — a bad edit of `main.go` during #838 removed the line and nothing noticed. Registered again, plus `TestEveryGroupedCommandIsRegistered`, which walks the help grouping | #842 |

## Where this plan deviated

- **`watch` shipped unreachable.** The guard that should have caught it did not exist: `watch` was in the help grouping, in the unit tests and in the module, and absent from the root command. The fix is one line and one test; the lesson is that "the help page lists it" was never asserted. #842 makes it an assertion.
- **A real `Idempotency-Key` is a column, not a table.** A key table needs pruning and a second transaction; a nullable unique column on the row it made gives the same promise with one index and dies with the row. Schema v5 refuses a v4 file with the reset instruction, like every version before it.
- **`describe` reads the running server**, never a copy embedded in the binary: a describe against a stale deployment should describe that deployment. The cost is that it needs the API up, which is the question it answers.
- **The verdict band is full width, not part of the aside.** On a narrow screen the aside comes after the evidence, and the verdict has to be the first thing read at every width; above 1024px it reads as a status band over the two columns.
- **MCP was declined.** The CLI stays the agent's primary interface; a thin MCP adapter over `internal/client` was considered and explicitly not built in this plan.
- **The console never asks DML or DDL.** Unchanged from the console epic, recorded here because the page was restructured around it.
