# [query]: run queries on Ctrl+Enter

- **Plan id:** issue 82
- **Area:** Changes & query
- **Status:** completed 2026-10-09
- **Issue:** [#82](https://github.com/dimasbaguspm/noxis/issues/82)
- **Pull request:** [#86](https://github.com/dimasbaguspm/noxis/pull/86)
- **Commit:** [5341667](https://github.com/dimasbaguspm/noxis/commit/5341667f123cb3b9f7af02e9bb3341b5deacccac) — `[ai:y][#82] feat: run queries on Ctrl+Enter (#86)`

## Plan

## What's problem?

The editor hint says "⌘/Ctrl + Enter" runs the query, but monaco has no binding for it, so the shortcut falls through to inserting a new line.

## What's affected and how?

- The SQL editor takes an optional run callback.
- A monaco keybinding maps Ctrl/Cmd+Enter to running the active query.
- The loading fallback textarea honors the same shortcut.

## Definition of Done

- Pressing Ctrl+Enter (and Cmd+Enter) in the query editor runs the query instead of inserting a new line.
- The hint stays accurate.

## Additional Notes

- None.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
