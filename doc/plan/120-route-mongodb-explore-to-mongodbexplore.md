# [console]: route mongodb explore to /mongodb/explore

- **Plan id:** issue 120
- **Area:** Console (web)
- **Status:** completed 2026-10-10
- **Issue:** [#120](https://github.com/dimasbaguspm/noxis/issues/120)
- **Pull request:** [#129](https://github.com/dimasbaguspm/noxis/pull/129)
- **Commit:** [48872fb](https://github.com/dimasbaguspm/noxis/commit/48872fbe85982a94edd36599eb299627985afecf) — `[ai:y][#120] feat(web): route mongodb explore to /mongodb/explore (#129)`

## Plan

## What's problem?

MongoDB explore lives at /mongodb while the family's other pages nest under /mongodb/requests — the route reads as a top level, not as part of the family.

## What's affected and how?

- Explorer moves to /mongodb/explore (route constant, router, sidebar nav via the family registry).
- /mongodb redirects to /mongodb/explore so old links and bookmarks keep working.
- Memcache stays at /memcache.

## Definition of Done

- The explore page is reachable at /mongodb/explore.
- Visiting /mongodb lands on the explore page (redirect).

## Additional Notes

- Blocked by nothing.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
