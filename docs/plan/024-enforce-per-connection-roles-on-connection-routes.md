# [api]: enforce per-connection roles on connection routes

- **Plan id:** issue 24
- **Area:** API & auth
- **Status:** completed 2026-10-08
- **Issue:** [#24](https://github.com/dimasbaguspm/noxis/issues/24)
- **Pull request:** [#32](https://github.com/dimasbaguspm/noxis/pull/32)
- **Commit:** [b0d2e7f](https://github.com/dimasbaguspm/noxis/commit/b0d2e7fb93454e4a269a0071c7a07c9f929dc7ce) — `[ai:y][#24] feat: enforce per-connection roles on connection routes (#32)`

## Plan

## What's problem?

With authentication and a grant store in place, every connection-scoped route must check the caller's effective level for that instance. A check that can be forgotten is worse than none.

## What's affected and how?

- Extend `controller.connections.raw(ctx, mod, name)` to `raw(ctx, mod, name, level)` so the single function that turns a name into a live handle is the one gate.
- List endpoints filter to connections the caller may `read`.
- Level map: read = list/get connection, schema, rows, query, test, keys, value, mongo databases/collections and read requests, list/get changes; write = alter, rollback, analyze, submit/batch change, mongo write requests; admin = approve, apply, reject, flush, delete keys.
- Connection management (`POST`/`DELETE /v1/{engine}/connections`) and `/internal/tick` stay global-admin-only.
- 403 with a clear message when the level is insufficient.

## Definition of Done

- [ ] Every connection-scoped handler passes a level to `raw`; no handler reaches a connection without a check.
- [ ] A `read` grant cannot alter, submit changes, or approve.
- [ ] A user sees only connections they can read.
- [ ] Global-admin-only routes reject non-admins with 403.
- [ ] Go build, lint and tests pass.

## Additional Notes

- Blocked by #22 (principal) and #23 (policy).
- No separate `approve` level: self-approval stays allowed on purpose.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
