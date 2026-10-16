# [console]: add the RBAC settings page for per-instance grants

- **Plan id:** issue 126
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#126](https://github.com/dimasbaguspm/noxis/issues/126)
- **Pull request:** [#135](https://github.com/dimasbaguspm/noxis/pull/135)
- **Commit:** [32f152d](https://github.com/dimasbaguspm/noxis/commit/32f152de3abadb94d419bc1b9212eed5aa5808f4) — `[ai:y][#126] feat(console): add the RBAC settings page (#135)`

## Plan

## What's problem?

Per-connection grants exist entirely server-side (connection_grants, enforced per request), but there is no console page to manage them, so only API calls can configure RBAC.

## What's affected and how?

- A Settings page lists every instance with its current grants (group → read/write/admin) and lets an admin add or remove grants.
- Group input is free-text with suggestions from the caller's own token groups, the configured admin group and groups already granted anywhere.
- The admin group remains implicit super-admin on every instance (unchanged, now documented in the UI).
- The existing grants endpoints stay as the transport.

## Definition of Done

- An admin can add a group with read/write/admin on an instance, change the level and remove it, from the Settings page.
- Non-admins cannot reach the page or the endpoints.
- The UI states which group is the implicit admin group.

## Additional Notes

- OIDC has no group directory over the standard endpoints; the page therefore builds its own RBAC vocabulary (free-text + known suggestions). An optional NOXIS_OIDC_GROUPS_API directory sync is a possible later extension.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
