# [plan]: harden the rehearsal and give the CLI an identity

Plan issue #762, closed 2026-11-20. Eight tickets, each its own branch and pull request.

## What problem does this solve?

The first pass proves the flow but leaves sharp edges: a reviewer can approve a change that the rehearsal could not scope (`CREATE INDEX`), the CLI has no identity of its own, the server has no observability, `apply` on a five-million-row table is a black box, and the acceptance suite depends on a hand-written docker compose. Each of those is a place where the tool asks to be trusted instead of showing its work.

## What is the user facing?

SQL that carries comments is accepted exactly as written. `CREATE INDEX` is rehearsed like anything else. `noxis auth signin <token>` stores a profile in the CLI's own dotconfig, and every boot asks the server who that token is before running anything. `noxis apply` reports progress on a large table instead of appearing to hang. Traces, logs and metrics go to whatever OTLP collector is configured. A package may not lose coverage without a review conversation about it.

## What shipped

| Ticket | What landed | PR |
| --- | --- | --- |
| #763 | Commented SQL classified (a `--` comment needs whitespace, per MySQL), `CREATE INDEX`/`DROP INDEX … ON t` scoped to their table | #771 |
| #764 | The acceptance engines boot through testcontainers; `compose.yml` and `seed.sql` are gone; three waits (port, root seed, `noxis` login) because a container answering ping is not one whose account works | #774 |
| #765 | Every `+` chain that read worse than a template became `fmt.Sprintf`, and the rule lives in AGENTS.md | #773 |
| #766 | `constants.ListenAddr`, `StatePath`, `ConsoleOrigins`; `HTTP_ADDR`, `STATE`, `CORS_ORIGINS` are gone; `noxisd` takes `-addr`/`-state` for development and the suite | #772 |
| #767 | `internal/common/telemetry`: traces, metrics and logs over OTLP, no-op with no endpoint; `http request` root span, one span per route, one per engine call; rehearsal and apply metrics | #775 |
| #768 | `Plan` sizes the change from `information_schema`; past 100k rows the claim happens in the request and the statement runs behind it (**202**), with `performance_schema` stages as progress and `INSTANT`/`INPLACE`/`COPY` named in the receipt; state file v4 | #776 |
| #769 | `internal/common/dotconfig` at 0600, `auth signin|status|signout`, `whoami` on boot cached for a minute, `--token` still works for CI and says once that it is the less private path | #777 |
| #770 | `test/coverage` + `floors.txt` in CI; domain to 96% | #779 |
| #778 | Use cases covered with handwritten port fakes (proposal 82, read 92, target 82, password 100), handlers 69, CLI 67 | #780, #781 |

## Where this plan deviated

- **`gocritic` does not enforce "Sprintf over `+`".** No honest lint rule exists for it (`perfsprint` argues the opposite), so the convention is documented in `AGENTS.md` and enforced by review. The epic's Done-when was written as if a linter would do it.
- **Spans are at the boundary, not inside every use case.** One span per request, one per route and one per engine call gives the trace a reviewer needs; per-`Run` spans in `internal/app` were not added because they buy nothing the route span does not show.
- **Coverage floors land where the tests land.** Each floor is today's number, raised in the pull request that earns it — a ratchet, not the 90/70/50 ceiling the plan imagined on day one. `internal/adapters/mysql` sits at 5% because its behaviour is proven by `@slow`, which is the plan's own instruction not to fake an engine.

## Risks that turned out to matter

- An instant `ALTER` finishes between two progress samples, so the watcher sees nothing. The receipt now compares `INNODB_TABLES.TOTAL_ROW_VERSIONS` before and after; a statement nobody can classify stays unknown rather than guessed.
- `information_schema.TABLES.TABLE_ROWS` is an estimate. It decides whether the answer waits, and the receipt carries it so a reviewer can see which number that decision used.
- A secret on disk is a trade: 0600, a 0600 directory, a 60-second re-check and a `signout` that deletes the profile — nothing more was claimed.

## Done when

- [x] A commented statement is classified, rehearsed and applied without stripping what the author wrote
- [x] `CREATE INDEX` scopes its table and rehearses to a diff
- [x] `make uat-slow` boots engines through testcontainers, with no compose file in the tree
- [x] No string is built with `+` where `fmt.Sprintf` reads better — as a convention, not a lint rule
- [x] The listen address and state path are constants, not environment variables
- [x] An OTLP collector receives traces, logs and metrics, and a span names its route or its engine call
- [x] Applying a change to a five-million-row table reports progress and names the algorithm the engine chose
- [x] `noxis auth signin`, `auth status` and `auth signout` work against the dotconfig, and every command fails early with a clear message when the token is not valid
