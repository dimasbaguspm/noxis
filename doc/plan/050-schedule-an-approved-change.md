# [changes]: schedule an approved change

- **Plan id:** issue 50
- **Area:** Changes & query
- **Status:** completed 2026-10-09
- **Issue:** [#50](https://github.com/dimasbaguspm/noxis/issues/50)
- **Pull request:** [#58](https://github.com/dimasbaguspm/noxis/pull/58)
- **Commit:** [7febd87](https://github.com/dimasbaguspm/noxis/commit/7febd87ca1ac6eeb0f964e74d163551fa8bdc6d5) — `[ai:y][#50] feat: schedule an approved change (#58)`

## Plan

## What's problem?

A change can only be scheduled when it is raised. Once it is approved there is no way to say "run this later", so a reviewed change still has to be applied by hand at the right moment.

## What's affected and how?

- An approved change gets a Schedule dropdown with Immediate and Schedule it.
- A new backend action sets or clears the schedule on an approved change.
- The scheduler applies the change when its slot passes, as it already does for scheduled submissions.

## Definition of Done

- Only an approved change shows the Schedule dropdown.
- Immediate clears any schedule so the change applies on Apply; Schedule it stores a future time and the scheduler applies it when due.
- A change that is not approved cannot be scheduled.

## Additional Notes

- Blocked by #49 (the create wizard no longer sets a schedule).

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
