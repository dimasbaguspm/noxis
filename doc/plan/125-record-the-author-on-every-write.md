# [console]: record the author on every write

- **Plan id:** issue 125
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#125](https://github.com/dimasbaguspm/noxis/issues/125)
- **Pull request:** [#134](https://github.com/dimasbaguspm/noxis/pull/134)
- **Commit:** [bf30af9](https://github.com/dimasbaguspm/noxis/commit/bf30af980e0a071208dcbb1df4e62da1f9eb48dd) — `[ai:y][#125] feat(changes): record the author on every write (#134)`

## Plan

## What's problem?

Nothing records who raised a change, submitted a document request or registered an instance, so the trail cannot answer "who did this?".

## What's affected and how?

- A createdBy field rides on SQL changes, MongoDB requests and instances, filled from the OIDC principal (display name, falling back to subject; empty in open mode).
- It appears on the change/request detail header and the instance record.
- The change detail page's audit trail gains an entry for the creation.

## Definition of Done

- Submitting a DML/DDL change, a MongoDB request or saving an instance stores the caller's identity.
- The identity is visible on the detail page header and in the audit trail.

## Additional Notes

- Blocked by nothing; backend + UAT + swagger + UI.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
