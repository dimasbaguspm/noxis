# [api]: schedule changes behind an internal tick

- **Plan id:** issue 11
- **Area:** API & auth
- **Status:** completed 2026-10-08
- **Issue:** [#11](https://github.com/dimasbaguspm/noxis/issues/11)
- **Pull request:** [#16](https://github.com/dimasbaguspm/noxis/pull/16)
- **Commit:** [0d0b6f4](https://github.com/dimasbaguspm/noxis/commit/0d0b6f49a1f112d939b427ba98db7a7e4c6d73ae) — `[ai:y][#11] feat: schedule changes behind an internal tick (#16)`

## Plan

## What's problem?

A change can only be applied by hand, immediately after approval. There is no way to say "execute this at 14:00", and the activity trail shows a "Scheduled" step that is not backed by anything (its own hint says "no scheduler yet").

## What's affected and how?

- A change carries a schedule: execute immediately (on apply, as today) or execute after a chosen time. The chosen time is stored on the change.
- The standalone binary runs an in-process scheduler, and the server also exposes `POST /internal/tick` so an external cron can drive it and it can be split into its own runtime later. Both call the same due-change handler.
- The tick endpoint is guarded by an API key header (new env).
- A tick applies approved changes whose time has passed; the applied result is recorded as usual. Overdue changes are picked up on the next tick after a restart.
- The activity trail shows "Scheduled" (with the chosen time) only when a schedule was set; immediate changes keep today's approve → apply trail.

## Definition of Done

- [ ] A change can carry "execute immediately" or "execute after `<datetime>`"; the decision and time are stored on the change.
- [ ] `POST /internal/tick` with a valid API key applies every approved change whose time has passed; a bad or missing key is rejected.
- [ ] The standalone binary schedules due changes without an external cron.
- [ ] Overdue changes run on the next tick after a restart.
- [ ] The review activity shows Scheduled (with the chosen time) only when a schedule was set.
- [ ] Existing unit + UAT suites pass.

## Additional Notes

- In-process: `robfig/cron/v3`. External: `cron`/`systemd` timer hitting `/internal/tick`. No queue dependency.
- Assumption: a change must be approved before its slot; a tick never applies an unapproved change.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
