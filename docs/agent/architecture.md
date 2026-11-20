# Architecture

## Layers

```
cmd/http ─┐                        transport: reads HTTP, calls one use case
cmd/cli  ─┤ (via internal/client)
          ▼
      internal/app                 use cases: the only place a workflow lives
          ▼
      internal/ports               interfaces the app needs, declared by the consumer
          ▼
      internal/domain              entities, state machine, sentinel errors — stdlib only

internal/adapters/sqlite ──▶ ports   implements the stores
internal/adapters/mysql  ──▶ ports   implements Driver for mysql and mariadb — one package,
                                     one DSN builder, the engine measured at probe time
```

Rules, enforced by `test/arch`:

1. `internal/domain` imports `internal/constants` (values only) and the standard library —
   no I/O, no context plumbing, no adapter.
2. `internal/ports` imports `domain` and the standard library.
3. `internal/app` imports `domain`, `ports`, pure helpers from `internal/common`, and at most
   one shared helper from a sibling package (`internal/app/password`, which opens a sealed
   credential for the use cases that connect). A use case never calls another use case's `Run`.
4. `internal/adapters/*` import `domain`, `ports`, `internal/common`, and their driver.
5. Only `cmd/*` names a concrete adapter.
6. Nothing imports `noxis/cmd/...` — hence the transports are both `package main`
   trees, and the CLI talks HTTP instead of linking a use case.

## Request flow

1. `middlewares.Chain` wraps the mux: `Recover → CORS → Access`. Access logs the
   status and the request id.
2. `routes.go` is the single route table — `{Method, Path, Handler, Scopes, Public}`.
   A route that is not `Public` gets `Authenticator.Require(scopes)` attached here, so
   scope rules are reviewable in one screen.
3. `Authenticator` resolves the bearer token against the env-declared set (constant
   time, over digests), attaches `domain.Caller{Name, Scopes}` to the request context,
   and refuses before the handler runs.
4. The handler reads the body and path params, calls one use case, and writes with
   `respond` / `fail`.
5. The use case calls the ports it holds as fields, built once in `cmd/http/wiring.go`,
   and returns a domain value or a sentinel error.
6. `fail` maps `ErrNotFound → 404`, `ErrForbidden → 403`, `ErrUnauthorized → 401`,
   `ErrConflict → 409`, `ErrUnsupported → 422`, `ErrInvalid → 400`; anything else is
   500 with a logged cause and a generic body.

## Identity

There is no user table. Callers are declared in `TOKENS`:

```json
{"agent": {"secret": "sk_...", "scopes": ["read", "propose"]},
 "ops":   {"secret": "sk_...", "scopes": ["read", "propose", "approve", "apply"]}}
```

The map key is the identity written into `proposed_by` / `decided_by` / `journal.actor`.
The secret is only ever compared as a SHA-256 digest, and never logged.

Separation of duties: `approve` and `apply` are separate scopes from `propose`, and the
decider must not be the proposer unless the decider's token holds `admin` — that
override is itself a journal row.

## References

`internal/common/ref` mints UUIDv7 in canonical form. Property relied on everywhere:
two refs sort by creation time, so `ORDER BY ref DESC` is both newest-first and a
stable keyset. A ref is validated on the way in (`ref.Valid`) — a path param that is
not a v7 ref is a 400, not a lookup.

## Adding a use case

1. `internal/domain` — the entity and its transitions. No I/O, no context plumbing.
2. `internal/ports` — the interface the use case needs, described in the use case's
   vocabulary (`Targets.ByRef`, not `SQLiteQuery`).
3. `internal/app/<verb>/run.go` — one struct with its dependencies as fields, one
   `Run(ctx, in) (out, error)`.
4. `internal/adapters/sqlite` — implement the port; assert it.
5. `cmd/http/handlers` — the handler; `cmd/http/routes.go` — the route with its scopes.
6. `test/uat/features/` — one scenario that proves the user-facing sentence.
