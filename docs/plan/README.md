# Plan archive

One file per completed `ai:y` plan, snapshotted from the tracker into this directory by
plan id. The README is the index.

**Convention:** a plan is one issue (or PR when no issue exists). The issue body is the
plan of record and follows
_What problem does this solve? → What is the user facing? → How will I solve it? → Done
when → Risks & notes_, matching `.github/ISSUE_TEMPLATE/ticket.md`. Write each
paragraph on one line; never hard-wrap the body. Each commit carries
`[ai:y][#<issue>] <type>(<scope>): <subject> (#<pr>)`.

The v1 plans are **not** carried over: they describe the tree on the `v1` branch
(thirteen engines, embedded console, dry-run verifier). Read them there if you need the
history of a decision. This index starts at the v2 rebuild.

## Index

### Foundation (v2)

- [[api]: the state file and target registration](750-the-state-file-and-target-registration.md) — plan issue #750, PR #755, merged 2026-11-20
- [[api]: probe a target and declare what its engine can do](751-probe-a-target-and-what-its-engine-can-do.md) — plan issue #751, PR #756, merged 2026-11-20
- [[api]: read through a target](752-read-through-a-target.md) — plan issue #752, PR #757, merged 2026-11-20
- [[chore]: wire the @slow acceptance harness](753-rehearsal-harness.md) — PR #758, merged 2026-11-20
- [[api]: rehearse a proposal](753-rehearse-a-proposal.md) — plan issue #753, PR #759, merged 2026-11-20
- [[api]: approve and apply a proposal](754-approve-and-apply.md) — plan issue #754, PR #760, merged 2026-11-20
- **[v2 epic: reviewed access for agents](749-reviewed-access-for-agents.md)** — plan issue #749, closed 2026-11-20
