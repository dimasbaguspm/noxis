# [backend] C4 restructure: extract application layer, rename controller to api

- **Plan id:** issue 166
- **Area:** Backend
- **Status:** completed 2026-10-10
- **Issue:** [#166](https://github.com/dimasbaguspm/noxis/issues/166)
- **Pull request:** [#167](https://github.com/dimasbaguspm/noxis/pull/167)
- **Commit:** [e58c86c](https://github.com/dimasbaguspm/noxis/commit/e58c86ca989637f0ef1ab450001c1d0838073485) — `[ai:y][#166] refactor(backend): split transport from application per C4 layering (#167)`

## Plan

## What's problem?
The  package mixes transport, workflow logic, authorization policy and identity handling in one layer (3456 LOC, 20 files), and dependencies are injected through the request context (Deps), so handlers are only testable over HTTP.

## What's affected and how?
All of cmd/noxis/server/controller/* and the unit tests + middleware.go + server.go wiring. Behavior must not change: API surface, swagger annotations, and responses stay identical; UAT is the guard.

## Definition of Done
- New internal/application package: use cases (changes/connections/grants workflows), policy (requireAdmin/authorize/canRead/effectiveLevel/actor/opLevel), identity (WithPrincipal/Principal).
- controller renamed to api (transport): thin handlers with swag annotations, Server{App, Auth, TickKey} replacing request-context Deps; pagination + module resolution stay in transport.
- Middleware stays in transport; WithPrincipal imported from application.
- Tests move with the code; official flow: go build/vet, unit tests, make swagger + pnpm gen:api unchanged, UAT green.

## Additional Notes
Big-bang refactor chosen deliberately. No behavior change; generated API docs must be byte-identical.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
