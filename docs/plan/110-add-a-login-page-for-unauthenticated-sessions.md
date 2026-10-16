# [console]: add a login page for unauthenticated sessions

- **Plan id:** issue 110
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#110](https://github.com/dimasbaguspm/noxis/issues/110)
- **Pull request:** [#114](https://github.com/dimasbaguspm/noxis/pull/114)
- **Commit:** [b87f539](https://github.com/dimasbaguspm/noxis/commit/b87f539d9bca70c6cb781fca28ddcc28e31d16f3) — `[ai:y][#110] feat: add a login page for unauthenticated sessions (#114)`

## Plan

## What's problem?

With no valid session the console bounces straight to the identity provider's sign-in page, so there is no console-branded entry point.

## What's affected and how?

- A /login route (outside the app shell) renders a sign-in card: product name, one-line pitch, and a single "Sign in" CTA that starts the OIDC flow.
- The auth provider stops auto-signing-in when there is no session; it marks the app unauthenticated and lets the router show /login.
- API 401 handling routes to /login instead of starting sign-in.
- The OIDC callback route keeps the code exchange; open mode (auth disabled) never shows login.

## Definition of Done

- Visiting the console with no session lands on /login without a detour to the IdP.
- The CTA goes through OIDC and the callback lands back in the shell.
- Expired sessions and 401s land on /login.
- Open mode is unaffected: straight into the shell.

## Additional Notes

- The login route lives outside the shell so it renders without chrome.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
