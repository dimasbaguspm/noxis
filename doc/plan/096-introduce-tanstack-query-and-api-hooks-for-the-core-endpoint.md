# [console]: introduce TanStack Query and api hooks for the core endpoints

- **Plan id:** issue 96
- **Area:** Console (web)
- **Status:** completed 2026-10-09
- **Issue:** [#96](https://github.com/dimasbaguspm/noxis/issues/96)
- **Pull request:** [#103](https://github.com/dimasbaguspm/noxis/pull/103)
- **Commit:** [38fb30c](https://github.com/dimasbaguspm/noxis/commit/38fb30cb19402c64eeeadaab6e3f480a0d1c5039) — `[ai:y][#96] feat: introduce TanStack Query and per-endpoint hooks (#103)`

## Plan

## What's problem?

Data is fetched with hand-rolled useState/useEffect code duplicated across pages and "custom hooks" that each re-implement loading/error/refetch, so there is no shared cache, invalidation, or retry story.

## What's affected and how?

- TanStack Query becomes the data core: a QueryClientProvider mounts at the app root.
- A dedicated query hook per endpoint appears for the core surfaces (instances, me, SQL schema/rows/run, changes list/detail and every change action), following a fixed naming: queries end in Query (e.g. usePagedInstancesQuery, useChangeListQuery), actions end in Mutation (e.g. useSubmitChangeMutation, useApplyChangeMutation).
- A query-key factory centralizes cache keys so invalidations stay consistent.
- The dashboard, instance detail, query editor page and the DML/DDL queues + detail stop managing their own lists/loading and instead read the hooks' state (isPending / isError / data).
- The existing typed api transport stays as-is underneath.

## Definition of Done

- Running the console fetches go through TanStack Query (network tab shows the same endpoints; no double-fetch storms).
- Every core list/detail/action above is served by a dedicated hook; no page still hand-fetches for these surfaces.
- Cached reads and invalidations behave (mutating a change refreshes the queue; a new instance appears on the dashboard).
- Errors surface via the shared UI pattern, not page-local toasts-in-useEffect.

## Additional Notes

- Blocked by nothing (B optional); lands before the later sweep tickets so pages are migrated once.
- Deliberate scope: memcache + mongodb surfaces migrate in a later ticket.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
