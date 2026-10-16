# [backend] wire shapes + identity into entity, AuthInfo into config

- **Plan id:** issue 176
- **Area:** Backend
- **Status:** completed 2026-10-10
- **Issue:** [#176](https://github.com/dimasbaguspm/noxis/issues/176)
- **Pull request:** [#178](https://github.com/dimasbaguspm/noxis/pull/178)
- **Commit:** [1ce5f81](https://github.com/dimasbaguspm/noxis/commit/1ce5f81d252ae0dff444683e3475e5e0036d5c02) — `[ai:y][#176] refactor(backend): wire shapes and identity into entity, AuthInfo into config (#178)`

## Plan

## What's problem?

Type placement and naming drift from the entity/interfaces chapter of the layer map: wire shapes live with the modules (`sql.Table/Column`, `memcache.Key/KeyPage`, `mongodb.Database/Collection/Request/Result`), `Principal` is named by usage not domain, and the public OIDC `AuthInfo` sits in `usecases` (transport config in the business layer).

## What's affected and how?

Structural move — behavior identical; swagger + web client regenerate once.

- **Wire shapes → entity**: `sql.Table/Column` → `entity.Table/Column`; `memcache.Key/KeyPage` → `entity.Key/KeyPage`; `mongodb.Database/Collection/Request` → `entity.Database/Collection/MongoRequest`.
- **Result merge**: `mongodb.Result` (embeds `entity.Result` + `Documents`) merges into the base `entity.Result` as `Documents []string` (omitempty); the mongo-specific type disappears, one shared result everywhere.
- **`entity.Principal` → `entity.Identity`** (name only — the authn identity from the issuer; fields unchanged). All refs: module/auth verify, middlewares, usecases identity, handlers MeResponse.
- **`AuthInfo` → `common/config`** (`config.AuthInfo`): moved out of `usecases.Deps` (field stays), produced by `newDeps`/config, consumed by `/auth/config`.
- Modules keep only adapter state (`DB`, `OIDC`); `interfaces` aliases re-point to entity types; method signatures updated.

## Definition of Done

- Swagger diff = renamed schema keys + `entity.Result.documents` only; web client regenerated + typecheck green.
- Unit tests + UAT + golangci-lint green.
- No `struct` declared inside a function anywhere in `internal/modules`.

## Additional Notes

- Churn cost is exactly the renamed swagger schema keys (`memcache.*`, `mongodb.*` → `entity.*`); that is the point of centralizing the wire contract.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
