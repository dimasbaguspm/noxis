# [web]: prefix console routes with the engine family and drive them from one family registry

- **Plan id:** issue 2
- **Area:** Console (web)
- **Status:** completed 2026-10-07
- **Issue:** [#2](https://github.com/dimasbaguspm/noxis/issues/2)
- **Pull request:** [#4](https://github.com/dimasbaguspm/noxis/pull/4)
- **Commit:** [0dc1eb1](https://github.com/dimasbaguspm/noxis/commit/0dc1eb18fb614831b1e81441737790d3a2e039fd) — `[ai:y][#2] refactor: prefix console routes with the engine family (#4)`

## Plan

## What's problem?

Console URLs do not say which engine family a page belongs to: SQL pages live at `/query`, `/dml`, `/ddl`, `/analytics`. The family split is also repeated in the router, the navigation registry and the shared connection state, and API calls hardcode each family's URL prefix. Both the URL and the wiring should be keyed by the engine family.

## What's affected and how?

- SQL pages move under `/sql` (query, dml, ddl, analytics). MongoDB and Memory Cache pages already carry their prefix and keep their paths. Dashboard stays `/`.
- Links, breadcrumbs, sidebar, header palette and the change/request detail pages point at the family-prefixed paths.
- Request URLs are built from the family, matching the backend's generic endpoint.
- Navigation and connection state come from one family registry, not three hardcoded lists.

## Definition of Done

- [ ] Every console page loads at its family-prefixed URL.
- [ ] The old unprefixed SQL paths redirect to their `/sql/` equivalents.
- [ ] No page or API call hardcodes a family URL prefix; all build it from the family.
- [ ] Breadcrumbs, sidebar, palette and in-page links resolve to the new paths.
- [ ] Package build/type-check passes and the console still embeds and serves.

## Additional Notes

- Blocked by #1.
- Dashboard and the new-connection wizard stay cross-family and read the registry.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
