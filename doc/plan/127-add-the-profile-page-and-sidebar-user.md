# [console]: add the profile page and sidebar user

- **Plan id:** issue 127
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#127](https://github.com/dimasbaguspm/noxis/issues/127)
- **Pull request:** [#136](https://github.com/dimasbaguspm/noxis/pull/136)
- **Commit:** [31f5565](https://github.com/dimasbaguspm/noxis/commit/31f5565b53fa90dbf51e0a282f36cecea13a0556) — `[ai:y][#127] feat(console): add the profile page and sidebar user (#136)`

## Plan

## What's problem?

The console knows who you are for authorization (subject + groups + admin) but never shows who you are: the signed-in user has no name, email or profile anywhere.

## What's affected and how?

- The OIDC principal gains display name and email (name/preferred_username/email claims), surfaced through /v1/me.
- A user card sits at the bottom of the sidebar (avatar initials, name, email, chevron) with a dropdown to the profile page and sign-out.
- A profile page shows identity details: name, email, subject, groups, admin flag and the effective read/write/admin level on every instance.

## Definition of Done

- The sidebar shows the signed-in user bottom-left; open mode shows a neutral "open mode" card.
- The profile page shows the identity and per-instance levels; sign-out works from the card.

## Additional Notes

- Blocked by nothing; touches claims parsing, swagger, api types, sidebar footer and a new page.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
