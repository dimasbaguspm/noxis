# [api]: authenticate callers with OIDC bearer tokens

- **Plan id:** issue 22
- **Area:** API & auth
- **Status:** completed 2026-10-08
- **Issue:** [#22](https://github.com/dimasbaguspm/noxis/issues/22)
- **Pull request:** [#30](https://github.com/dimasbaguspm/noxis/pull/30)
- **Commit:** [84bbcc8](https://github.com/dimasbaguspm/noxis/commit/84bbcc81fd152b529a29a05eae32798e5a88831a) — `[ai:y][#22] feat: authenticate callers with OIDC bearer tokens (#30)`

## Plan

## What's problem?

The API has no authentication: every route is open and CORS allows any origin. We want to rely on Authentik for identity and group membership rather than building a user store.

## What's affected and how?

- Validate `Authorization: Bearer <jwt>` against the issuer's JWKS (verify `iss`, `aud`, `exp`, RS256). Use `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2`; no client secret (bearer-only, public client).
- New config, read once in `entity.Config` like the rest: `NOXIS_OIDC_ISSUER` (empty -> run open, loud warning), `NOXIS_OIDC_CLIENT_ID`, `NOXIS_OIDC_AUDIENCE` (default = client id), `NOXIS_OIDC_GROUPS_CLAIM` (default `groups`).
- Authenticate middleware puts `Principal{sub, groups}` in the request context; unauthenticated requests to `/v1/*` get 401.
- `GET /v1/me` returns `{sub, groups, admin}` so the console has one source of truth.
- Add `Authorization` to `Access-Control-Allow-Headers`.

## Definition of Done

- [ ] With `NOXIS_OIDC_ISSUER` set, a request without a valid token gets 401; with a valid token it is served.
- [ ] `iss`/`aud`/`exp`/signature are all validated; a token from another issuer or audience is rejected.
- [ ] `NOXIS_OIDC_ISSUER` empty keeps today's open behaviour with a warning at startup.
- [ ] `GET /v1/me` returns the principal from the presented token.
- [ ] Go build, lint and tests pass.

## Additional Notes

- Bearer-only on purpose: the server never handles cookies, sessions or refresh tokens; the console refreshes against Authentik directly.
- Verify Authentik's `aud` value (app slug vs client id) against a real token before hardcoding a default.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
