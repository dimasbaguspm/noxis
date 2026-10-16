# [web]: OIDC login and permission-gated console

- **Plan id:** issue 25
- **Area:** Console (web)
- **Status:** completed 2026-10-08
- **Issue:** [#25](https://github.com/dimasbaguspm/noxis/issues/25)
- **Pull request:** [#34](https://github.com/dimasbaguspm/noxis/pull/34)
- **Commit:** [23faba5](https://github.com/dimasbaguspm/noxis/commit/23faba5da41742784e46e4e4058d544b58b1bd8b) — `[ai:y][#25] feat: OIDC login and permission-gated console (#34)`

## Plan

## What's problem?

The console has no login and no notion of who the operator is. With the API requiring bearer tokens, the SPA must authenticate against Authentik and only offer actions the caller is allowed to take.

## What's affected and how?

- Add `oidc-client-ts` for Authorization Code + PKCE (public client), silent refresh with a short-lived access token.
- Attach `Authorization: Bearer <token>` in the single `request()` wrapper in `lib/api.ts`.
- Handle 401 by starting/redirecting to login and 403 by surfacing the error instead of failing silently.
- Read `GET /v1/me` for the principal and per-connection levels; gate navigation entries, the new-connection wizard, and write/admin actions (alter, submit, approve, flush, delete keys).
- The server remains the enforcement point; UI gating is cosmetic only.

## Definition of Done

- [ ] Unauthenticated use lands on Authentik login; after login the console loads.
- [ ] Access token is attached to every API call and refreshed before expiry.
- [ ] Actions above the caller's level are hidden or disabled.
- [ ] A 401 triggers re-login; a 403 shows a readable message.
- [ ] `pnpm typecheck`, `pnpm lint`, `pnpm build` pass.

## Additional Notes

- Blocked by #22 (`/v1/me`) and #24 (levels).
- Token storage: `localStorage` with a short access TTL is accepted for an operator tool; XSS tradeoff noted.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
