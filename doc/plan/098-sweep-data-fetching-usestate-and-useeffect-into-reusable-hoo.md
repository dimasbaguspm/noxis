# [console]: sweep data-fetching useState and useEffect into reusable hooks

- **Plan id:** issue 98
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#98](https://github.com/dimasbaguspm/noxis/issues/98)
- **Pull request:** [#105](https://github.com/dimasbaguspm/noxis/pull/105)
- **Commit:** [16bd849](https://github.com/dimasbaguspm/noxis/commit/16bd849b9e2bad5bd9edcfa51dc37d3b90e065ce) — `[ai:y][#98] refactor: replace the registry provider with query hooks (#105)`

## Plan

## What's problem?

Pages still declare data state (arrays, loading, refetch) that the query layer already owns, and several pages pile up raw useState calls (one page has 10) plus effects that trigger work, which is exactly what the new data layer should own instead.

## What's affected and how?

- No page owns list/loading/error state for server data anymore; everything reads query-hook state.
- Effects that trigger fetches or refetches are removed; actions invalidate the cache instead.
- The global per-family connection registry is replaced by the instances query + a small active-connection hook (persists per session, pickers keep working unchanged).
- Remaining useState is UI-only (tabs, expand/collapse, search text) and moves into small reusable hooks where it repeats (e.g. debounced search, active connection, dialog open state).

## Definition of Done

- No page declares useState for server data; no useEffect triggers a fetch or refetch.
- The registry provider is gone or reduced to the active-connection hook; connection pickers everywhere still work.
- Pages that previously used 5+ useState calls use the reusable hooks instead.
- Behavior unchanged (verified against the UAT suite).

## Additional Notes

- Blocked by the core and memcache/mongodb query tickets.

- Blocked by #96, #97.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
