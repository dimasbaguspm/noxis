# Architecture & patterns

Detail for `AGENTS.md`. Read that first for the map.

## C4-ish layering

```
transport   apps/api/cmd/noxis/server         (route table → use case Serve)
application apps/api/cmd/noxis/server/app.go   (composition root: newApp)
            apps/api/internal/usecases        (one file per use case; Serve(w, r) + core)
domain      apps/api/internal/entity             (data model, imports nothing internal)
            apps/api/internal/interfaces          (seams)
infra       apps/api/internal/modules/*          (engine adapters)
            apps/api/internal/repositories/*     (persistence)
cross-cut   apps/api/internal/common/*           (config, logging, idgen, pagination, secret)
            apps/api/internal/constants          (universal values)
```

Rule: dependencies point inward and only downward. The order (low → high) is
`constants · entity · interfaces → common/* → repositories/* → modules/*
(families) → modules (registry) → usecases → middlewares → cmd/noxis/server
(routes)`. A package may import only packages below its layer, and two packages
in the same layer may not import each other. `cmd/noxis` is the composition root
and is exempt. `test/arch/arch_test.go` reads the graph from `go list` and fails
on any bent edge, so the rule is enforced, not just documented. Repositories and
modules keep their `var _ interfaces.X` satisfier asserts next to the struct.

## Dependency injection

One file does all wiring, at boot, once: `apps/api/cmd/noxis/server/app.go`.

- `sources` holds the seams: the repositories, `modules.Registry`, `config.AuthInfo`.
- `app` lists every use case as a named field.
- `newApp(sources)` builds each use case **explicitly** — the fields it needs are
  named at the call site. There is no reflection and no dependency container in a
  request context; the only thing on the context is the authenticated principal.

Add a new use case: create `apps/api/internal/usecases/foo.go` with a `Foo struct`,
its `Run` core and a `Serve(w, r)` → add a field to `app` and one line to `newApp`
→ register its `Serve` in `newHandler`. No constructor boilerplate, no reflection.

## Engine registry

`apps/api/internal/modules/modules.go`:

- `Registry` resolves a family by name (`Engine`), by interface (`EngineSQL`,
  `EngineMongo`, `EngineMemcache`), and a driver → family (`Serves`).
- `Default()` builds `{sql, memcache, mongodb}`. **This map is the only place
  families are declared.**
- The registry is an explicit dependency: a use case that resolves a family
  embeds `usecases.RegistryDeps`, so `uc.Modules` is set at the composition root.

Serve methods take `{engine}` as a path param and resolve the module; a family that
does not serve the route gets `interfaces.ErrUnsupported` → 404, never the wrong
module. This is why one handler serves all families instead of a copy per family.

## Interfaces (the seams)

| Interface | File | Implemented by |
| --- | --- | --- |
| `ConnectionRepository` | `interfaces/repository.go` | `sqlite.Store` |
| `ChangeRepository` | `interfaces/repository.go` | `sqlite.Store` |
| `PermissionRepository` | `interfaces/repository.go` | `sqlite.Store` |
| `Module` / `SQLModule` | `interfaces/sql.go` | `sql.DB` |
| `MemcacheModule` | `interfaces/memcache.go` | `memcache.DB` |
| `MongoModule` | `interfaces/mongodb.go` | `mongodb.DB` |
| `Verifier` | `interfaces/auth.go` | `auth.OIDC` (tests use a fake) |

`ErrNotFound` and `ErrUnsupported` live in `interfaces` and drive HTTP 404.

## Use-case pattern

- One file, one exported struct, one business flow. The domain work is a
  `Run(ctx, ...)` core; its `Serve(w, r)` owns that route's HTTP.
- Fields are the deps it needs (filled by type). It references **no other use
  case**; shared steps are package-level helpers that take deps as args:
  `rawConn` (load connection + check family + authorize), `Authorize` /
  `RequireAdmin` / `EffectiveLevel` (policy), `applyChange`/`applySQL`/
  `applyMongo` (apply mechanics).
- Every `Run` opens a span: `ctx, end := logging.StartSpan(ctx); defer func(){ end(err) }()`.
- Domain rejects are returned, not panicked. `ErrForbidden` → 403.

## Serve pattern

`apps/api/internal/usecases/http.go` holds `respond`, `respondNoContent`,
`decode` and `fail` (the single error→status map). A `Serve` body resolves the
engine from the ctx registry, decodes, calls the domain core and writes:

```go
func (uc Foo) Serve(w http.ResponseWriter, r *http.Request) {
    mod, err := uc.Modules.Engine(r.PathValue("engine"))
    if err != nil { fail(w, err); return }
    out, err := uc.Run(r.Context(), mod, /* path/query/body */)
    if err != nil { fail(w, err); return }
    respond(w, out)
}
```

Routes are declared in `apps/api/cmd/noxis/server/server.go` (`newHandler`) and
point straight at a use case `Serve`: there is no handler package. Add a route
plus a swag annotation on `Serve`, then `make swagger`.

## Auth & policy

- `middlewares.WithAuth(verifier, next)` verifies bearer tokens; the
  `entity.Identity` lands on the ctx (`usecases.Principal`).
- Open mode is gone: `WithAuth` denies when no verifier is configured.
- `usecases.rawConn` is the single per-connection gate. `RequireAdmin` gates the
  management surfaces (connection CRUD, permissions).
- Levels: `read < write < admin` (`entity.LevelAllows`). `OpLevel` maps a SQL op
  to a level (only `query` is read; `analyze` mutates planner stats = write).
- List endpoints hide unreadable connections (`CanRead`/`readableConnections`).

## Span naming

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
must never leak into the memcache/mongo namespaces.

**New persisted field:** add to `entity`, add a **new** `ALTER TABLE` line to the
`migrations` slice in `sqlite.go`, update the `Save`/scan in the relevant
`apps/api/internal/repositories/sqlite/*.go` file. Never edit an existing migration.
