# [instances]: give connections a uuid and open a detail page

- **Plan id:** issue 40
- **Area:** Instances
- **Status:** completed 2026-10-09
- **Issue:** [#40](https://github.com/dimasbaguspm/noxis/issues/40)
- **Pull request:** [#44](https://github.com/dimasbaguspm/noxis/pull/44)
- **Commit:** [4b8fdc7](https://github.com/dimasbaguspm/noxis/commit/4b8fdc7fc5cd6656e54116cce4e9525bcecda84b) — `[ai:y][#40] feat: give connections a uuid and open a detail page (#44)`

## Plan

## What's problem?

Instance detail is a modal addressed by connection name. There is no stable identifier, so detail cannot move to a URL or grow, and the two families can only be told apart by which list a row came from.

## What's affected and how?

- Connections gain a stable uuid; existing rows are backfilled on startup.
- New cross-family read endpoints: look up one instance by id, and list one instance's changes.
- The dashboard "Detail" action becomes a link to a page instead of opening a modal.
- The page shows instance identity (driver, endpoint, database, user, family, registered) and its changes grouped (DML / DDL / status for SQL, requests for MongoDB; memory cache has none).
- Test and delete move onto the page.

## Definition of Done

- A pre-existing database starts with every connection holding a unique id.
- Clicking Detail on the dashboard lands on a page at `/instances/<id>`.
- The page shows identity plus changes grouped by kind/status.
- A caller without read access to an instance gets not-found, not the instance.
- Family-scoped routes, grants and changes that link by name still work.

## Additional Notes

- Password encryption at rest is out of scope (separate ticket).
- Renaming a connection is out of scope; name stays the stable link for grants and changes.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
