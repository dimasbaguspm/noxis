# [api]: rename the SQL analytics op to analyze

- **Plan id:** issue 21
- **Area:** API & auth
- **Status:** completed 2026-10-08
- **Issue:** [#21](https://github.com/dimasbaguspm/noxis/issues/21)
- **Pull request:** [#29](https://github.com/dimasbaguspm/noxis/pull/29)
- **Commit:** [cf87bf0](https://github.com/dimasbaguspm/noxis/commit/cf87bf0c030018a90e6eefdc6600a9fa39c19fae) — `[ai:y][#21] refactor: rename the SQL analytics op to analyze (#29)`

## Plan

## What's problem?

`POST /v1/sql/connections/{name}/analytics` runs the SQL `ANALYZE` statement to refresh planner statistics. The route, op constant and client method all call it "analytics", which collides with the removed metrics feature and misdescribes what it does.

## What's affected and how?

- Route `/analytics` -> `/analyze` (handler `PostAnalytics` -> `PostAnalyze`).
- `constants.OpAnalytics` -> `OpAnalyze`.
- Frontend `api.ops.analytics` -> `api.ops.analyze` and the call site in `pages/query.tsx`.
- Regenerate swagger and `web/src/api/generated.ts`.

## Definition of Done

- [ ] No `analytics` naming remains for this endpoint in API or console.
- [ ] `make swagger` regenerates; `generated.ts` matches.
- [ ] The Query page "Analyze" action still refreshes planner stats.
- [ ] Go and web checks pass.

## Additional Notes

- RBAC note: this mutates the target's planner stats, so it is a **write** operation.
- Independent of the SSE removal, but lands before RBAC so the level map is final.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
