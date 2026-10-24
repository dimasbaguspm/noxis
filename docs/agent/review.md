# Code review checklist

Detail for `AGENTS.md`. Use this when reviewing a diff or a PR. It is
repo-specific: the generic review skill covers the rest.

## Invariants (a diff that breaks one is a blocker)

1. **Layer direction** — nothing inward imports outward. `entity` imports
   nothing internal; `usecases` never import `handlers`; repositories never
   import use cases. New seams go in `apps/api/internal/interfaces`.
2. **Access gate** — every connection-scoped flow must go through
   `usecases.RegistryDeps.RawConn` (family check + `Authorize`), `RequireSuper` for the
   root-only surfaces or `RequireAdmin` for the instance-management surfaces. A
   new route that reads a connection without one of these is a security hole.
   `RequireAdmin` never implies data access: an admin without a grant cannot read
   rows.
3. **Password redaction** — the wire never carries `entity.Connection.Password`.
   Use `Connection.Redacted()`. Check every new response shape.
4. **Family isolation** — a memcache connection must be invisible to SQL routes
   and vice versa. Handlers resolve `{engine}` via the registry; no driver →
   family switch in code.
5. **Changes are artifacts** — never delete a `changes` row; a failed
   submission is returned to the caller, not persisted (see `SubmitChange`).
6. **Migrations are append-only** — a new column is a new `ALTER TABLE` line;
   never edit or reorder existing ones.
7. **SQL guard** — statement-kind checks run **per statement** (split outside
   string literals/comments). A read endpoint must reject DML/DDL smuggled
   behind `SELECT 1; DROP ...`.

## What to verify per layer

**Handlers** — thin: decode → resolve module → call use case → return. No
business logic, no direct repository access, no engine-family literal. Swag
annotation block updated and `make swagger` run if the route changed.

**Use cases (tests)** — a use-case test drives `Serve` through `internal/common/testkit` and states its own expectations on the generated mocks in `internal/mocks` (`make mocks` after an interface changes). Persistence behaviour is tested against real SQLite, not mocks.

**Use cases** — one file, one `Serve(w, r)` + `Run` core, deps as explicit struct fields. `Serve` opens its own span and passes `ctx` to `uc.Fail`; a decision that changes the outcome is recorded with `logging.InfoWithCtx`/`WarnWithCtx`/`ErrorWithCtx`. Load registered capabilities through `common.LoadModule(ctx, &target)`; route-specific engines must attach the path family with `common.WithModuleFamily`, and missing or unsupported modules must fail the request. Keep repositories and other app dependencies out of context.
Shared logic lives in a package helper, not by calling another use case. Errors
wrapped with `%w`; sentinel compares with `errors.Is`.

**Modules** — owns the domain rules for its area (`sql.Changes` and
`mongodb.Changes` are the change lifecycle). A rule inside a `Serve` that is not
about HTTP is a finding: it belongs in the module. Implements an `interfaces.*Module`; has the `var _ interfaces.X`
satisfier; no engine-specific literal outside `apps/api/internal/constants`; times out
every network call.

**Repositories** — new method added to both the interface and `Store`;
`ErrNotFound` for missing rows; transactions via `WithTx`; no env reads.

**Wire shapes** — a request body or response envelope is a struct in
`entity/api.go`; a `map[string]any` response is a review blocker. `pnpm lint` + `pnpm build` pass; API types come from
`src/api/generated.ts` (run `gen:api` if the spec changed); no hardcoded driver
list that drifts from `GET /external/v1/config/drivers`.

## Conventions a reviewer should enforce

- Terse, purposeful comments: explain *why*, not *what*. `ponytail:` comments
  mark a known limit (`shortcut: <limit>, <when to upgrade>`) — either fine to
  keep or a reason to ask.
- No new dependency for a few lines.
- No option/config/abstraction nobody asked for.
- Errors lowercase, no punctuation (revive `error-strings`).
- `context.Context` is the first arg (`context-as-argument`).
- Every `http.Client`/DB call has a timeout.
- Constants over magic strings; `apps/api/internal/constants` for universal values.

## Verification (run before approving)

```bash
make lint     # golangci-lint v2: errcheck/govet/staticcheck/revive/gocritic/...
make test     # unit
make build    # bin/noxis
make web      # console build + embed sync (if apps/web changed)
make uat      # if the REST surface / engine behaviour changed
```

CI additionally builds both images (`Dockerfile`, `Dockerfile.api`) and runs
the web lint + typecheck + build.

## Commit / PR conventions (observed in history)

- Commit subject: `[ai:y][#<issue>] <type>(<scope>): <subject>` where `<type>`
  is Conventional Commits (`feat`/`refactor`/`fix`/`chore`) and scope is
  `backend`/`web`.
- One PR = one issue; the PR body should state scope + what was verified.
- When asked to open a PR, follow the `mr` skill; for commits, the
  `git-commit` skill.

## Retired design — do not restore

A master/slave + Kafka + Docker topology was the original design and is fully
retired. Any branch, issue or note that mentions `doorman`, `slave`, `kafka`,
`franz`, `NOXIS_BROKER` or `NOXIS_TOPIC_PREFIX` predates the rewrite: the tree
is one SQLite-backed binary with no broker and no docker socket. A diff that
"restores" those is wrong.

## Definition of done

- [ ] `make lint` + `make test` green locally.
- [ ] REST surface changed ⇒ `make swagger` run and `openapi.*` committed.
- [ ] UI changed ⇒ `make web` (or `pnpm build`) green.
- [ ] Auth/engine behaviour changed ⇒ `make uat` green.
- [ ] New persisted column ⇒ appended migration + interface/Store updated.
- [ ] Access gate present on any new connection-scoped route.
- [ ] No password on the wire; no secret logged.
