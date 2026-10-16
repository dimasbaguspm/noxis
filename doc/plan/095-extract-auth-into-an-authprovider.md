# [console]: extract auth into an AuthProvider

- **Plan id:** issue 95
- **Area:** Console (web)
- **Status:** completed 2026-10-09
- **Issue:** [#95](https://github.com/dimasbaguspm/noxis/issues/95)
- **Pull request:** [#102](https://github.com/dimasbaguspm/noxis/pull/102)
- **Commit:** [412f572](https://github.com/dimasbaguspm/noxis/commit/412f572f35b527e31e88096894ba36b5ce183f19) — `[ai:y][#95] refactor: extract auth into an AuthProvider (#102)`

## Plan

## What's problem?

Auth boot (OIDC config, redirect, silent renew, identity, permission checks, sign-out) is tangled into the same provider that owns the connection registry, drivers and schema, so auth cannot be reasoned about or reused on its own.

## What's affected and how?

- A dedicated AuthProvider owns: bootstrapping OIDC, the "redirecting to sign-in" flow, current identity, enabled/admin flags, the can() permission helper, and sign-out.
- The rest of the app consumes it through a small useAuth() hook.
- The connection registry and schema provider stop knowing anything about auth.
- The OIDC callback page keeps working unchanged.

## Definition of Done

- Auth state and helpers are only served by the AuthProvider / useAuth().
- No auth code (config load, user manager, identity, signout) remains in the registry/schema provider.
- Open mode (auth disabled) and enabled mode both boot exactly as before.
- The 401 "redirecting to sign-in" behavior still works.

## Additional Notes

- Blocked by nothing; start anytime.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
