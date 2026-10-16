# [backend] usecases stand alone: drop Base, explicit deps, no cross-use-case imports

- **Plan id:** issue 185
- **Area:** Backend
- **Status:** completed 2026-10-10
- **Issue:** [#185](https://github.com/dimasbaguspm/noxis/issues/185)
- **Pull request:** [#186](https://github.com/dimasbaguspm/noxis/pull/186)
- **Commit:** [baa5c10](https://github.com/dimasbaguspm/noxis/commit/baa5c101bda8fbe2266fcf60c38539b22d034c04) — `[ai:y][#185] refactor(backend): usecases stand alone — no Base, explicit deps, shared helpers (#186)`

## Plan

## What's problem?

The use-case-per-file restructure kept a shared `Base` struct; every use case embeds it, which re-creates a god-object (all four deps always present) and lets the tick reach other use cases through it. Use cases should stand alone: own injected deps, no sibling imports.

## What's affected and how?

- **`usecase.go` deleted**; `Base` gone.
- Each use case struct declares only the repository/module fields its flow needs (`SubmitChange{Connections, Changes, Grants}`, `Approve{Changes, Grants}`, `Tick{Connections, Changes, Grants, Modules}`, …) and takes them via construction, not embedding.
- Shared logic becomes package-level helper functions (same rules as policy/identity): `raw`, `authorize/canRead/effectiveLevel/readableConnections`, `moduleByDriver`, and the apply machinery (`applySQL`/`applyMongo`) which `Apply`, `ApplyMongo` and `Tick` all call with their own deps — **no use case imports another use case**.
- Gateways keep the seven handler seams, hold the same explicit dep fields, and construct the per-op use cases.
- Tests rewire construction from `Base` structs to explicit field bundles.

## Definition of Done

- `grep` verification: no `Base` anywhere; no use-case file references another use-case type.
- Handlers unchanged (seams identical); zero swagger/web delta.
- Unit + UAT + lint green.

## Additional Notes

- `deps.go`/`identity.go`/helpぃers stay package-level (context plumbing, policy, apply) — they are not use cases.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
