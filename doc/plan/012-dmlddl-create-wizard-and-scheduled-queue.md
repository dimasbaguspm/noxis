# [web]: DML/DDL create wizard and scheduled queue

- **Plan id:** issue 12
- **Area:** Console (web)
- **Status:** completed 2026-10-08
- **Issue:** [#12](https://github.com/dimasbaguspm/noxis/issues/12)
- **Pull request:** [#17](https://github.com/dimasbaguspm/noxis/pull/17)
- **Commit:** [d4226fd](https://github.com/dimasbaguspm/noxis/commit/d4226fd90e4164cc7425e0870f73265fa753cbdf) — `[ai:y][#12] feat: DML/DDL create wizard and scheduled queue (#17)`

## Plan

## What's problem?

Creating a DML/DDL change is a single modal with no guidance, there is no choice of when it runs, and the DML/DDL queue mixes immediate and scheduled work together.

## What's affected and how?

- A create wizard for DML and DDL: instance → kind → original query → rollback query → schedule, keeping the statement-kind badge.
- The schedule step chooses "execute immediately" or "execute after `<datetime>`" — the decision the activity trail later reflects.
- The DML/DDL queue shows estimated rows/duration before execution, a DML/DDL breakdown, and separate Immediate vs Scheduled groups.

## Definition of Done

- [ ] Raising a DML/DDL change goes through a step-by-step wizard instead of one modal.
- [ ] The wizard sets execute-immediately or execute-after-`<datetime>`.
- [ ] The queue shows estimates before execution and a DML/DDL breakdown.
- [ ] The queue groups Immediate and Scheduled changes separately.
- [ ] `pnpm typecheck`, `pnpm lint`, `pnpm build` pass.

## Additional Notes

- Blocked by #11 (the schedule field).
- Reuses #10 (file attach) inside the wizard if it has landed.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
