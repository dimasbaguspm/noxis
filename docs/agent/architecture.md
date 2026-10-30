# Architecture & patterns

Detail for `AGENTS.md`. Read that first for the map.

## C4-ish layering

```
transport   apps/api/cmd/noxis/server         (route table → use case Serve)
application apps/api/cmd/noxis/server/app.go   (composition root: newApp)
            apps/api/internal/usecases        (root framework; one subpackage per domain)
domain      apps/api/internal/entity             (data model, imports nothing internal)
            apps/api/internal/interfaces          (seams)
infra       apps/api/internal/modules/*          (engine adapters, one package per driver)
            apps/api/internal/repositories/*     (persistence)
cross-cut   apps/api/internal/common/*           (config, logging, idgen, pagination, secret, shared changeflow rules)
            apps/api/internal/constants          (universal values)
```

Rule: dependencies point inward and only downward. The order (low → high) is
`constants · entity · interfaces → common/* → repositories/* → modules/<driver>
→ modules (registry) → usecases (root) → usecases/* (domains) → middlewares → cmd/noxis/server
(routes)`. A package may import only packages below its layer, and two packages
in the same layer may not import each other. `cmd/noxis` is the composition root
and is exempt. `test/arch/arch_test.go` reads the graph from `go list` and fails
on any bent edge, so the rule is enforced, not just documented. Repositories and
modules keep their `var _ interfaces.X` satisfier asserts next to the struct.

## Dependency injection

One file does all wiring, at boot, once: `apps/api/cmd/noxis/server/app.go`.

- `sources` holds the seams: `repositories.Registry`, `modules.Registry`, and `config.AuthInfo`.
- `app` lists every use case as a named field.
- `newApp(sources)` builds each use case **explicitly** — the fields it needs are
  named at the call site, and repositories stay under `Deps.Repositories`.
  The request context carries the authenticated principal
  and immutable module loader only; repositories, configuration, and the rest of
  the app dependencies remain explicit.
- `server.Run` attaches the loader to the server base context once. A use case loads
  a typed capability with `common.LoadModule(ctx, &target)`; route-specific engine
  use cases first attach the `{driver}` path value with `common.WithModuleDriver`.
  Generic lookups use `Modules.Engine(driver)`.

Add a new use case: create `apps/api/internal/usecases/<domain>/foo.go` with a
`Foo struct` embedding `usecases.RegistryDeps` and a `Serve(w, r)` → add a field
to `app` and one line to `newApp` → add a `route` row with its method, path and
access rule. No constructor boilerplate, no reflection.

## Engine registry

`apps/api/internal/modules/modules.go`:

- `Registry` resolves a driver name to its module (`Engine`); typed lookups go through `common.ModuleLoader`, not one method per capability.
- `Default(repositories, metrics)` builds one module per driver — `postgres`, `mysql`,
  `mariadb`, `sqlite`, `cockroach`, `redis`, `valkey`, `mongo` — plus each driver's
  change flow. **This map is the only place the process declares what it can do**,
  and a map hit can never depend on iteration order.
- Each driver is its own package under `apps/api/internal/modules/<driver>`, holding
  its own connection string, quote style, catalog queries, proof strategy and
  verification rules. Driver packages share nothing but `interfaces` and
  `constants`: no shared kit, and no driver package imports another. Duplication is
  a decision; `test/driver/conformance_test.go` is the guard.
- The registry implements the common typed `ModuleLoader` for registered engine and
  change-flow capabilities. `common.LoadModule(ctx, &target)` resolves them from
  request context; engine lookups use the driver the route attached, and absent or
  unsupported targets return `constants.ErrUnsupported`.
- The registry is an explicit dependency: a use case that resolves a driver
  embeds `usecases.RegistryDeps`, so `uc.Modules` is set at the composition root.

Serve methods take `{driver}` as a path param and resolve the module; a driver that
does not serve the route gets `constants.ErrUnsupported` → 404, and a connection that
belongs to another driver is refused by name. This is why one handler serves every
driver instead of a copy per driver.

## Interfaces (the seams)

| Interface | File | Implemented by |
| --- | --- | --- |
| `ConnectionRepository` | `interfaces/connection.go` | `sqlite.Store` |
| `ChangeRepository` | `interfaces/change.go` | `sqlite.Store` |
| `PermissionRepository` | `interfaces/permission.go` | `sqlite.Store` |
| `Module` + capabilities (`ConsoleQuerier`, `SchemaReader`, `ServerReporter`, `ChangeProver`, `ChangeApplier`) | `interfaces/sql.go` | each `modules/<sql driver>.DB` (`Tables` carries declared keys, so the schema browser and the ERD read one pass) |
| `KeyBrowser` | `interfaces/memcache.go` | `redis.DB`, `valkey.DB` |
| `DocumentBrowser`, `MongoChangeEngine` | `interfaces/mongodb.go` | `mongodb.DB` |
| `Verifier` | `interfaces/auth.go` | `auth.OIDC` (tests use a fake) |
| `SQLChangeFlow` / `MongoChangeFlow` | `interfaces/changeflow.go` | every `modules/<driver>.Changes` |

A driver implements only the capabilities it can prove, and a compile-time
`var _ interfaces.X` assert per capability keeps the claim honest — the loader
returns unsupported rather than a stub.

`ErrNotFound` and `ErrUnsupported` live in `constants` and drive HTTP 404.

## Where the rules live

A use case authorizes the caller, decodes the request, maps the answer to a
status, and nothing else. Domain rules live in a module: the change lifecycle
belongs to the driver that runs it, so each `modules/<driver>.Changes` owns
submit → validate → approve/reject/schedule → apply. Their shared state rules come
from `internal/common/changeflow`, a base that all of them embed — it is a library,
not a registered module. It sits below every driver package in the enforced
dependency graph; engine execution and decision logging stay with the driver
modules.

Every queue is registered in `modules.Default()` next to its engine. Use cases
load `interfaces.SQLChangeFlow` or `interfaces.MongoChangeFlow` through
`common.LoadModule`; the SQL, document and cache capabilities use the same loader
with the route's driver attached. A module may read repositories; it never imports
the registry or a sibling driver package.

## Use-case pattern

- The root package `internal/usecases` is the framework: `RegistryDeps` (named
  `Repositories` bundle plus the module registry) and its methods. Domain subpackages
  (`connection`, `change`, `apply`, `schema`, `permission`, `instance`, `auth`,
  `cache`, `mongo`, `driver`) hold the use cases; siblings never import each other.
- One exported struct per operation, one `Serve(w, r)`. It embeds
  `usecases.RegistryDeps`, so the shared steps are methods on the receiver:
  `uc.RawConn` (load connection + check driver + authorize), `uc.Authorize` /
  `uc.RequireSuper` / `uc.RequireAdmin` / `uc.EffectiveGrant` (policy),
  `uc.Respond` / `uc.Fail` / `uc.Decode` (transport). `applyChange`/`applySQL`/`applyMongo` live in `apply`.
- Domain rejects are returned, not panicked. `constants.ErrForbidden` → 403.

## Serve pattern

`apps/api/internal/usecases/http.go` holds `Respond`, `RespondNoContent`,
`Decode` and `Fail` (the single error→status map) as methods on `RegistryDeps`.
A `Serve` resolves the engine, decodes, does the work and writes:

```go
func (uc Foo) Serve(w http.ResponseWriter, r *http.Request) {
    ctx, end := logging.StartSpan(r.Context())
    defer end(nil)
    mod, err := uc.Modules.Engine(r.PathValue("driver"))
    if err != nil { uc.Fail(ctx, w, err); return }
    out, err := doWork(ctx, mod, /* path/query/body */)
    if err != nil { uc.Fail(ctx, w, err); return }
    uc.Respond(w, out)
}
```

Routes are declared in `apps/api/cmd/noxis/server/routes.go` (`routes` +
`newHandler`) and point straight at a use case `Serve`; there is no handler package
or use-case middleware wrapper. Add a route plus a swag annotation on `Serve`, then
`make swagger`.

Every request body and response envelope is a struct in
`apps/api/internal/entity/api.go` — a use case never answers with a
`map[string]any`, so the OpenAPI schema and the generated web client come from
the same declaration a reader can find.

## Auth & policy

- `middlewares.WithAuth(verifier, next)` verifies bearer tokens; the
  `entity.Identity` lands on the context (`entity.Principal`).
- Open mode is gone: `WithAuth` denies when no verifier is configured.
- `usecases.RegistryDeps.RawConn` is the single per-connection gate. Two global gates
  sit on
  top of it: `RequireSuper` for root-only surfaces (connection CRUD, instance
  credentials, role changes) and `RequireAdmin` for the instance-management
  surfaces (user directory, account status, every connection's grant policy,
  access-request decisions).
- Three global roles (`entity.RoleSuper/Admin/Member`). Exactly one account is
  `super`: the env root named by `NOXIS_ADMIN_EMAIL`, promoted on every boot. A
  `member` still needs a grant per connection, and an `admin` role never widens
  data access.
- An OIDC login whose verified email equals an account's `email` is blended into
  it (`internal/auth.Service.Verify` → `GetUserByEmail`): same subject, role and
  grants. Unmatched or unverified OIDC identities are recorded as reference rows,
  then resolved from SQLite on every request; stored role/status are authoritative
  and banned identities are refused.
- Levels: `read < write < admin` (`entity.LevelAllows`). `OpLevel` maps a SQL op
  to a level (only `query` is read; `analyze` mutates planner stats = write).
- List endpoints hide unreadable connections (`CanRead`/`readableConnections`).
  The one exception is `GET /external/v1/access` (`permission.Matrix`): it is
  metadata only (names, drivers, grants) and exists so an admin can see the whole
  access landscape without a grant on each connection.

## Test doubles

The authenticated caller rides the request context via `entity.WithPrincipal` /
`entity.Principal`, so every layer above entity reads it the same way. Structured
logs derive `user_id`, `user_role`, and groups from that one principal.

`apps/api/internal/mocks` holds `go.uber.org/mock` doubles for the repository and
engine-module seams, generated from `internal/interfaces` with `make mocks`.
`apps/api/internal/common/testkit` holds the two helpers a use-case test needs to
send one request; it holds no fakes. A test
declares only the calls it expects, and persistence is covered against real
SQLite (`sqlite/users_test.go`, `modules/sqlite/sqlite_test.go`).

## Tracing & span naming

One span per request, opened by `middlewares.Trace` **outside** `WithAuth` so a
request refused before the router is still traceable, and closed with an error
when the handler answered 5xx. `middlewares.RequestLog` runs inside the auth
middleware and attaches the caller to that span once.

Every use case opens its own operation span at the top of `Serve`
(`ctx, end := logging.StartSpan(r.Context()); defer end(nil)`), so the trace names
the operation (`Usecases.Change.Submit.Serve`) and the use case owns its own
conditions. Failures are recorded once, by the exit that handles them:
`uc.Fail(ctx, w, err)` maps the error to a status and calls
`logging.ErrorWithCtx`, which puts the error and the failed status on the span.

Logging is the three-level API in `common/logging`: `InfoWithCtx` for a state
change, `WarnWithCtx` for an expected refusal, `ErrorWithCtx(ctx, msg, err, ...)`
for a failure — each writes the span event and the log record together, so a
trace and a log file tell the same story.

`apps/api/internal/common/logging/span.go` derives span names from the Go function:
`Repository.Sqlite.Store.List`, `Usecases.Op.Serve`, `Modules.Postgres.DB.query`.
Logs re-emit trace/span ids from ctx so log/trace/metric share identity
(`logging` package).

## How to add things

**New driver:** create `apps/api/internal/modules/<driver>/`, implement
`interfaces.Module` plus only the capability interfaces the server can honour,
register it in `modules.Default()` under its own driver name, add the driver const
and its capability consts, add its row to the driver catalog
(`usecases/driver/driver.go`), and add a compose service plus an acceptance
scenario. The driver name is the API namespace, so no route edit is needed.

**New SQL-looking operation:** add the op const, add the method to the capability
interface in `interfaces/sql.go`, implement it in every SQL driver package that can
honour it, add a use case (or extend `usecases.Op` if it fits the op map), add a
route + swag. A driver that cannot honour it does not implement the interface, and
the loader refuses the capability by name instead of a stub returning "not
available". A batch that runs statements goes through `runBatch`
(`modules/<driver>/sql.go`): one transaction, commit unless the caller is a dry
run, so apply and dry-run never diverge.

**New persisted field:** update the canonical SQLite schema and repository `Save`/scan, bump the schema version, and update schema tests. Existing stores are not migrated; startup rejects older versions and requires an intentional reset.
