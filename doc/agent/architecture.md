# Architecture & patterns

Detail for `AGENTS.md`. Read that first for the map.

## C4-ish layering

```
transport   cmd/noxis/server/handlers   (plain funcs, swag, JSON glue)
application cmd/noxis/server/deps        (DI container: Deps → Uses)
            internal/usecases           (one file per use case)
domain      internal/entity             (data model, imports nothing internal)
            internal/interfaces          (seams)
infra       internal/modules/*          (engine adapters)
            internal/repositories/*     (persistence)
cross-cut   internal/common/*           (config, logging, idgen, pagination, secret)
            internal/constants          (universal values)
```

Rule: dependencies point inward. `handlers → usecases → modules/repositories → entity`.
`entity` imports nothing internal. Repositories/modules own the `var _ interfaces.X`
satisfier asserts next to their struct.

## Dependency injection

Two files do all wiring, at boot, once.

- `cmd/noxis/server/deps/deps.go` — `Deps` holds the **sources**: 3 repositories,
  `modules.Registry`, `*slog.Logger`, `config.AuthInfo`.
- `deps/uses.go` — `Uses` lists every bound use case as a named field; `Init(d)`
  is the single construction point.
- `deps/inject.go` — `WithDeps[T](d, target)` reflects over the target's fields
  and fills each by **field type** from `Deps`. A missing dependency stays zero
  and fails loudly on use.
- `deps.Seed(ctx, Uses)` / `deps.From(ctx)` — the uses ride the request context;
  `deps.From` panics if unseeded (wiring bug).

Add a new use case: create `internal/usecases/foo.go` with a `Foo struct` and a
`Serve` method → add one field to `Uses` and one line to `Init` → call it from a
handler. No constructor boilerplate.

## Engine registry

`internal/modules/modules.go`:

- `Registry` resolves a family by name (`Engine`), by interface (`EngineSQL`,
  `EngineMongo`, `EngineMemcache`), and a driver → family (`Serves`).
- `Default()` builds `{sql, memcache, mongodb}`. **This map is the only place
  families are declared.**
- The registry is put on the request context at boot (`modules.WithRegistry`),
  so any use case can resolve an engine from its ctx.
- `RegistryFrom(ctx)` panics if unseeded.

Handlers take `{engine}` as a path param and resolve the module; a family that
does not serve the route gets `interfaces.ErrUnsupported` → 404, never the wrong
module. This is why one handler serves all families instead of a copy per family.

## Interfaces (the seams)

| Interface | File | Implemented by |
| --- | --- | --- |
| `ConnectionRepository` | `interfaces/repository.go` | `sqlite.Store` |
| `ChangeRepository` | `interfaces/repository.go` | `sqlite.Store` |
| `GrantRepository` | `interfaces/repository.go` | `sqlite.Store` |
| `Module` / `SQLModule` | `interfaces/sql.go` | `sql.DB` |
| `MemcacheModule` | `interfaces/memcache.go` | `memcache.DB` |
| `MongoModule` | `interfaces/mongodb.go` | `mongodb.DB` |
| `Verifier` | `interfaces/auth.go` | `auth.OIDC` (tests use a fake) |

`ErrNotFound` and `ErrUnsupported` live in `interfaces` and drive HTTP 404.

## Use-case pattern

- One file, one exported struct, one `Serve(ctx, ...)` method.
- Fields are the deps it needs (filled by type). It references **no other use
  case**; shared steps are package-level helpers that take deps as args:
  `rawConn` (load connection + check family + authorize), `Authorize` /
  `RequireAdmin` / `EffectiveLevel` (policy), `applyChange`/`applySQL`/
  `applyMongo` (apply mechanics).
- Every `Serve` opens a span: `ctx, end := logging.StartSpan(ctx); defer func(){ end(err) }()`.
- Domain rejects are returned, not panicked. `ErrForbidden` → 403.

## Handler pattern

`cmd/noxis/server/handlers/server.go` holds `handle`, `handleNoContent`,
`decode`, `writeJSON`, `writeErr`. A handler body is:

```go
func Foo(w http.ResponseWriter, r *http.Request) {
    d := deps.From(r.Context())
    handle(w, r, func(ctx context.Context) (any, error) {
        mod, err := d.Modules.Engine(r.PathValue("engine"))
        if err != nil { return nil, err }
        return d.Foo.Serve(ctx, mod, /* path/query/body */)
    })
}
```

Routes are declared in `cmd/noxis/server/server.go` (`newHandler`). Add a route
there + a handler + a swag annotation block, then `make swagger`.

## Auth & policy

- `middlewares.WithAuth(verifier, next)` verifies bearer tokens; the
  `entity.Identity` lands on the ctx (`usecases.Principal`).
- Open mode = no issuer set; every caller trusted; `deps.From` still required.
- `usecases.rawConn` is the single per-connection gate. `RequireAdmin` gates the
  management surfaces (connection CRUD, grants).
- Levels: `read < write < admin` (`entity.LevelAllows`). `OpLevel` maps a SQL op
  to a level (only `query` is read; `analyze` mutates planner stats = write).
- List endpoints hide unreadable connections (`CanRead`/`readableConnections`).

## Span naming

`internal/common/logging/span.go` derives span names from the Go function:
`Repository.Sqlite.Store.List`, `Usecases.Op.Serve`, `Modules.Sql.DB.query`.
Logs re-emit trace/span ids from ctx so log/trace/metric share identity
(`logging` package).

## How to add things

**New engine family:** implement `interfaces.Module` (+ one of the sub-seam
interfaces) in `internal/modules/<family>/`, register in `modules.Default()`,
add a family const + driver consts, add a route family in `newHandler`, add
`Engine<Family>` to `Registry`. Add a compose service to exercise it in UAT.

**New SQL-looking operation:** add the op const, add a method to `SQLModule`,
implement it in `modules/sql`, add a use case (or extend `usecases.Op` if it fits
the op map), add a handler + route + swag. Keep the family split: SQL-only ops
must never leak into the memcache/mongo namespaces.

**New persisted field:** add to `entity`, add a **new** `ALTER TABLE` line to the
`migrations` slice in `sqlite.go`, update the `Save`/scan in the relevant
`internal/repositories/sqlite/*.go` file. Never edit an existing migration.
