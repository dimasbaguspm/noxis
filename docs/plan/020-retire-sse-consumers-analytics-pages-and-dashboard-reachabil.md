# [web]: retire SSE consumers, analytics pages and dashboard reachability

- **Plan id:** issue 20
- **Area:** Console (web)
- **Status:** completed 2026-10-08
- **Issue:** [#20](https://github.com/dimasbaguspm/noxis/issues/20)
- **Pull request:** [#28](https://github.com/dimasbaguspm/noxis/pull/28)
- **Commit:** [4687efc](https://github.com/dimasbaguspm/noxis/commit/4687efcf0add57bd777b0b824f46a5a741e71a93) — `[ai:y][#20] refactor: retire SSE consumers, analytics pages and dashboard reachability (#28)`

## Plan

## What's problem?

The console subscribes to SSE in four hooks, renders three Analytics pages fed by those streams, and shows per-row reachability on the dashboard from the same stream. The endpoints are being removed (see #19), so every consumer goes with them.

## What's affected and how?

- Delete pages `analytics.tsx`, `memcache-analytics.tsx`, `mongodb-analytics.tsx`.
- Remove the three `ANALYTICS` entries from `lib/routes.ts`, the three nav entries + `BarChart3` import from `lib/families.ts`, and the three lazy imports + routes from `main.tsx`.
- Remove the `EventSource` blocks from `hooks/use-data.ts`, `hooks/use-memcache.ts`, `hooks/use-mongodb.ts` (keep the rest of each hook; trace every consumer of the stats they expose).
- Delete `hooks/use-connection-states.ts` and the dashboard reachability column (`states`/`connected`/`StateCell` in `pages/dashboard.tsx`). Dashboard tile counts (All/SQL/Memory Cache/MongoDB) stay — they read the connection list.
- Remove `statsStream` (×3) and `statusStream` from `lib/api.ts`.
- Remove `recharts` from `package.json`.

## Definition of Done

- [ ] No `EventSource` or `recharts` reference remains in `web/src`.
- [ ] The dashboard renders without a reachability column and without a live-connection indicator.
- [ ] `pnpm typecheck`, `pnpm lint`, `pnpm build` pass.
- [ ] No orphaned imports or dead hooks left behind.

## Additional Notes

- Blocked by #19 (the routes must be gone first).
- The detail dialog may read the reachability map; grep every consumer before cutting.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
