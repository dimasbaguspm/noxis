# [console]: filter the dashboard through the instances API

- **Plan id:** issue 53
- **Area:** Console (web)
- **Status:** completed 2026-10-09
- **Issue:** [#53](https://github.com/dimasbaguspm/noxis/issues/53)
- **Pull request:** [#55](https://github.com/dimasbaguspm/noxis/pull/55)
- **Commit:** [1a8fc03](https://github.com/dimasbaguspm/noxis/commit/1a8fc0361c917ce79a1cfc49938fc756cd9e8040) — `[ai:y][#53] feat: filter the dashboard through the instances API (#55)`

## Plan

## What's problem?

The dashboard fetches every instance once and filters them in the browser, even though the API accepts name and driver filters.

## What's affected and how?

- The name box and driver dropdown call `GET /v1/instances` with name and driver filters; the table renders the server result.
- The shared registry still loads for the other pages.

## Definition of Done

- Typing a name or picking a driver issues an API request and the table shows the response.
- Clearing the filters restores the full list.
- A caller still only sees instances they can read.

## Additional Notes

- The instance list endpoint already supports the filters; this is a console change.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
