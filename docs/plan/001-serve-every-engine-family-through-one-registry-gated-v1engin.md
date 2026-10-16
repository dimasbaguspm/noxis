# [api]: serve every engine family through one registry-gated /v1/{engine} route table

- **Plan id:** issue 1
- **Area:** API & auth
- **Status:** completed 2026-10-07
- **Issue:** [#1](https://github.com/dimasbaguspm/noxis/issues/1)
- **Pull request:** [#3](https://github.com/dimasbaguspm/noxis/pull/3)
- **Commit:** [163dc35](https://github.com/dimasbaguspm/noxis/commit/163dc3504074e9b68247f827f756e5428abd37de) — `[ai:y][#1] refactor: gate engine families behind one route registry (#3)`

## Plan

## What's problem?

The API knows each engine family by name. Routes are registered per family by hand, the handler dependencies carry one typed field per family, and resolving a driver to its family is a switch statement. Adding or regating a family means editing several shared places, and no single endpoint is generic over the engine.

## What's affected and how?

- The HTTP surface becomes one route table over `/v1/{engine}/...`, where `{engine}` is `sql`, `mongodb` or `memcache`. Existing URLs and response bodies are unchanged; only registration and dispatch become generic.
- A sub-resource an engine does not serve (keys on SQL, changes on a memory cache) is refused as not-found instead of reaching the wrong module.
- Driver-to-family resolution goes through the registered modules, not a hardcoded driver switch.
- Family-only handlers resolve the module from the path parameter and check the capability they need; cross-family handlers stay shared.

## Definition of Done

- [ ] build, lint and the existing test suite pass.
- [ ] Families are registered in one place; the driver-to-family switch is gone.
- [ ] Routes are registered once per path with an `{engine}` parameter; none duplicated per family.
- [ ] Every README curl example returns the same status and body as before.
- [ ] Unknown engine, or an unsupported sub-resource, returns 404 (never 500).
- [ ] swagger regenerates cleanly and documents `{engine}`.

## Additional Notes

- Split from the console ticket; the console is blocked by this.
- Decision: one generic endpoint only, no per-family aliases.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
