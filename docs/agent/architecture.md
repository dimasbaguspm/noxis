# Architecture & patterns

Detail for `AGENTS.md`. Read that first for the map.

## C4-ish layering

```
transport   apps/api/cmd/noxis/server         (route table → use case Serve)
application apps/api/cmd/noxis/server/app.go   (composition root: newApp)
            apps/api/internal/usecases        (root framework; one subpackage per domain)
domain      apps/api/internal/entity             (data model, imports nothing internal)
            apps/api/internal/interfaces          (seams)
infra       apps/api/internal/modules/*          (engine adapters)
            apps/api/internal/repositories/*     (persistence)
cross-cut   apps/api/internal/common/*           (config, logging, idgen, pagination, secret, shared changeflow rules)
            apps/api/internal/constants          (universal values)
```

Rule: dependencies point inward and only downward. The order (low → high) is
`constants · entity · interfaces → common/* → repositories/* → modules/*
(families) → modules (registry) → usecases (root) → usecases/* (domains) → middlewares → cmd/noxis/server
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
  use cases first attach `{engine}` with `common.WithModuleFamily`. Generic engine
  checks continue to use `Modules.Engine(name)`.

Add a new use case: create `apps/api/internal/usecases/<domain>/foo.go` with a
`Foo struct` embedding `usecases.RegistryDeps` and a `Serve(w, r)` → add a field
to `app` and one line to `newApp` → register its `Serve` in `newHandler`. No
constructor boilerplate, no reflection.

## Engine registry

`apps/api/internal/modules/modules.go`:

- `Registry` resolves a family by name (`Engine`), by interface (`EngineSQL`,
  `EngineMongo`, `EngineMemcache`), and a driver → family (`Serves`).
- `Default(connections, changes, metrics, auth)` builds `{sql, memcache, mongodb}` and their change flows. **This map
  is the only place families are declared.**
- `Registry` implements the common typed `ModuleLoader` for registered engine and
  change-flow capabilities. `common.LoadModule(ctx, &target)` resolves them from
  request context; engine lookups use the family attached by the route, and absent
  or unsupported targets return an error.
- The registry is an explicit dependency: a use case that resolves a family
  embeds `usecases.RegistryDeps`, so `uc.Modules` is set at the composition root.

Serve methods take `{engine}` as a path param and resolve the module; a family that
does not serve the route gets `constants.ErrUnsupported` → 404, never the wrong
module. This is why one handler serves all families instead of a copy per family.

## Interfaces (the seams)

| Interface | File | Implemented by |
| --- | --- | --- |
| `ConnectionRepository` | `interfaces/connection.go` | `sqlite.Store` |
| `ChangeRepository` | `interfaces/change.go` | `sqlite.Store` |
| `PermissionRepository` | `interfaces/permission.go` | `sqlite.Store` |
| `Module` / `SQLModule` | `interfaces/sql.go` | `sql.DB` (`Tables` carries declared keys, so the schema browser and the ERD read one pass) |
| `MemcacheModule` | `interfaces/memcache.go` | `memcache.DB` |
| `MongoModule` | `interfaces/mongodb.go` | `mongodb.DB` |
| `Verifier` | `interfaces/auth.go` | `auth.OIDC` (tests use a fake) |
| `SQLChangeFlow` / `MongoChangeFlow` | `interfaces/change.go` | `sql.Changes`, `mongodb.Changes` |

`ErrNotFound` and `ErrUnsupported` live in `constants` and drive HTTP 404.

## Where the rules live

A use case authorizes the caller, decodes the request, maps the answer to a
status, and nothing else. Domain rules live in a module: the change lifecycle
belongs to the family that runs it, so `modules/sql.Changes` and
`modules/mongodb.Changes` each own submit → validate → approve/reject/schedule →
apply. Their shared state rules come from `internal/common/changeflow`, a base
that both embed — it is a library, not a registered module. It sits below both
family modules in the enforced dependency graph; engine execution and decision
logging stay with the family modules.

Both queues are registered in `modules.Default()` next to the engines. Use cases
load `interfaces.SQLChangeFlow` or `interfaces.MongoChangeFlow` through
`common.LoadModule`; SQL, Mongo, and cache engine capabilities use the same loader
with the route family attached. A module may read repositories; it never imports
the registry or a sibling family package.

## Use-case pattern

- The root package `internal/usecases` is the framework: `RegistryDeps` (named
  `Repositories` bundle plus the module registry) and its methods. Domain subpackages
  (`connection`, `change`, `apply`, `schema`, `permission`, `instance`, `auth`,
  `cache`, `mongo`, `driver`) hold the use cases; siblings never import each other.
- One exported struct per operation, one `Serve(w, r)`. It embeds
  `usecases.RegistryDeps`, so the shared steps are methods on the receiver:
  `uc.RawConn` (load connection + check family + authorize), `uc.Authorize` /
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
    mod, err := uc.Modules.Engine(r.PathValue("engine"))
    if err != nil { uc.Fail(ctx, w, err); return }
    out, err := doWork(ctx, mod, /* path/query/body */)
    if err != nil { uc.Fail(ctx, w, err); return }
    uc.Respond(w, out)
}
```

Routes are declared in `apps/api/cmd/noxis/server/server.go` (`newHandler`) and
point straight at a use case `Serve`; there is no handler package or use-case
middleware wrapper. Add a route plus a swag annotation on `Serve`, then
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
  it (`modules/auth.Service.Verify` → `GetUserByEmail`): same subject, role and
  grants. Unmatched or unverified OIDC identities are recorded as reference rows.
- Levels: `read < write < admin` (`entity.LevelAllows`). `OpLevel` maps a SQL op
  to a level (only `query` is read; `analyze` mutates planner stats = write).
- List endpoints hide unreadable connections (`CanRead`/`readableConnections`).
  The one exception is `GET /external/v1/access` (`permission.Matrix`): it is
  metadata only (names, drivers, grants) and exists so an admin can see the whole
  access landscape without a grant on each connection.

## Test doubles

The authenticated caller rides the request context via `entity.WithPrincipal` /
`entity.Principal`, so every layer above entity reads it the same way.

`apps/api/internal/mocks` holds `go.uber.org/mock` doubles for the repository and
engine-module seams, generated from `internal/interfaces` with `make mocks`.
`apps/api/internal/common/testkit` holds the two helpers a use-case test needs to
send one request; it holds no fakes. A test
declares only the calls it expects, and persistence is covered against real
SQLite (`sqlite/users_test.go`, `modules/sql/sqlite_test.go`).

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
`Repository.Sqlite.Store.List`, `Usecases.Op.Serve`, `Modules.Sql.DB.query`.
Logs re-emit trace/span ids from ctx so log/trace/metric share identity
(`logging` package).

## How to add things

**New engine family:** implement `interfaces.Module` (+ one of the sub-seam
interfaces) in `apps/api/internal/modules/<family>/`, register in `modules.Default()`,
add a family const + driver consts, add a route family in `newHandler`, add
`Engine<Family>` to `Registry`. Add a compose service to exercise it in UAT.

**New SQL-looking operation:** add the op const, add a method to `SQLModule`,
implement it in `modules/sql`, add a use case (or extend `usecases.Op` if it fits
the op map), add a handler + route + swag. Keep the family split: SQL-only ops
must never leak into the memcache/mongo namespaces. A batch that runs statements
goes through `runBatch` (`modules/sql/sql.go`): one transaction, commit unless the
caller is a dry run, so apply and dry-run never diverge.

**New persisted field:** add to `entity`, add a **new** `ALTER TABLE` line to the
`migrations` slice in `sqlite.go`, update the `Save`/scan in the relevant
`apps/api/internal/repositories/sqlite/*.go` file. Never edit an existing migration.
