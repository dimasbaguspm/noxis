# [api]: store per-connection role grants

- **Plan id:** issue 23
- **Area:** API & auth
- **Status:** completed 2026-10-08
- **Issue:** [#23](https://github.com/dimasbaguspm/noxis/issues/23)
- **Pull request:** [#31](https://github.com/dimasbaguspm/noxis/pull/31)
- **Commit:** [0b9544e](https://github.com/dimasbaguspm/noxis/commit/0b9544eb47ab390dba27986b6397b29c820730b5) — `[ai:y][#23] feat: store per-connection role grants (#31)`

## Plan

## What's problem?

Noxis must decide what a group may do on each registered instance, without owning identities. There is no policy store today.

## What's affected and how?

- New table `connection_grants(connection, group_name, level)`, unique on `(connection, group_name)`. Default deny.
- Three levels: `read` < `write` < `admin`.
- Repository methods to list/set/delete grants for a connection, behind an interface like the existing ones.
- A global-admin-only CRUD endpoint (or endpoints) to manage grants.
- Global admin is implicit: members of `NOXIS_OIDC_ADMIN_GROUP` — no row needed. They bootstrap the first grant.

## Definition of Done

- [ ] Grants persist in SQLite and survive restart.
- [ ] Effective level = max across the caller's groups; unknown group = deny.
- [ ] Only global admins can read or write grants; others get 403.
- [ ] Migration is additive and idempotent like the existing ones.
- [ ] Go build, lint and tests pass.

## Additional Notes

- This stores policy (group -> instance -> level), never identity or credentials.
- No per-user rows; group membership is read from the token on every request.
- Blocked by #22 for the admin principal, but the table/CRUD can land behind it.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
