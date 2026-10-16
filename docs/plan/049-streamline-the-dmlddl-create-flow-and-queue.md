# [changes]: streamline the DML/DDL create flow and queue

- **Plan id:** issue 49
- **Area:** Changes & query
- **Status:** completed 2026-10-09
- **Issue:** [#49](https://github.com/dimasbaguspm/noxis/issues/49)
- **Pull request:** [#57](https://github.com/dimasbaguspm/noxis/pull/57)
- **Commit:** [147f0f7](https://github.com/dimasbaguspm/noxis/commit/147f0f78458a1197aec6cfbdb6e96df58429431e) — `[ai:y][#49] feat: streamline the DML/DDL create flow and queue (#57)`

## Plan

## What's problem?

Raising a change takes five steps, two of which the server does not need: the kind is re-derived from the statement, and the schedule belongs to the approval step. The queue also splits immediate and scheduled work into two lists that are otherwise identical.

## What's affected and how?

- The create wizard asks for instance, notes, original query and rollback query, then submits; the kind and schedule steps are removed.
- The modal is wider.
- The DML and DDL queues render one list instead of immediate + scheduled.

## Definition of Done

- The wizard has no kind or schedule controls and submits after the query and rollback are entered.
- A submitted change is validated and recorded as before.
- Each queue is a single list; no Immediate/Scheduled split remains.

## Additional Notes

- Scheduling moves to the approved change (separate ticket, blocked by this one).

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
