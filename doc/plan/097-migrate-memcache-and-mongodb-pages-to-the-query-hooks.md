# [console]: migrate memcache and mongodb pages to the query hooks

- **Plan id:** issue 97
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#97](https://github.com/dimasbaguspm/noxis/issues/97)
- **Pull request:** [#104](https://github.com/dimasbaguspm/noxis/pull/104)
- **Commit:** [f9d5940](https://github.com/dimasbaguspm/noxis/commit/f9d5940065471dd506cad7a89a2b7077009107fc) — `[ai:y][#97] feat: move memcache and mongodb pages to the query hooks (#104)`

## Plan

## What's problem?

The memcache and mongodb pages still fetch with their own hooks and useState/useEffect, so two data models coexist with the new query-hook pattern.

## What's affected and how?

- Memcache and mongodb pages move onto the same dedicated-query-hook pattern as the core surfaces.
- The standalone memcache/mongodb data hooks disappear (their jobs move into per-endpoint query hooks).
- Keys, flush, and mongodb requests keep their exact behavior.

## Definition of Done

- Memcache keys/value/flush and mongodb browse + request list/detail/new all run through the query hooks.
- No memcache/mongodb page hand-fetches or owns list/loading state.
- The removed hooks have no remaining importers.

## Additional Notes

- Blocked by the core query-hooks ticket.

- Blocked by #96.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
