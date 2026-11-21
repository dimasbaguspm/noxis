---
name: noxis-review
description: Decide on a noXis proposal, apply it, and report what the rehearsal proved. Use when a change is waiting for approve/reject/apply — hold this skill only with an approve+apply token, never the one that proposed.
---

# noXis review

A proposal is decided by a **different token holder** than the one that proposed it. If you hold
`approve`/`apply` and also wrote the proposal, you are reviewing yourself — the one thing this product
exists to prevent.

## 1. Read the queue

```bash
noxis proposals --status verified --json          # rehearsed, waiting for you
noxis proposals --target <ref> --limit 50 --json
noxis show-proposal <pr> --json                   # the change, the rollback, the rehearsal
noxis events <pr> --json                          # who proposed it, when, and every step since
```

## 2. Judge the evidence, not the prose

| Read | Ask |
| --- | --- |
| `tier` | `shadow`/`rehearsal`: it ran somewhere real. `static`: **nothing ran** — the note names what to grant |
| `schema_diff` | is this the shape you expected? does it keep defaults and nullability? |
| `row_diff` | for DML: how many rows, and are they the rows you meant? |
| `rollback` | does it actually undo the statement? the rehearsal already ran it and diffed the schema back |
| `note` | the proposer's reason, which is not evidence |
| estimate / locks | a rewrite on a table of 40M rows takes a lock; say so before you approve it |

A `static` proposal is not proof. Approving one is a decision to trust the syntax and the plan, and it
should be said out loud.

## 3. Decide

```bash
noxis approve <pr> --note "diff matches the plan; rollout after 22:00 UTC"
noxis reject  <pr> --note "needs a backfill in the same change"
```

- The note is the record. Write the reason a person in an incident review would want to read.
- Rejecting is not archiving: the proposal stays, with its evidence, forever. A better change is a new
  proposal.

## 4. Apply

```bash
noxis apply <pr> --wait --json
noxis apply <pr> --confirm-destructive --wait     # a change with no rollback, explicitly acknowledged
```

- Apply runs the statement against the target. A claim is made **before** the statement, so a crash leaves
  `applied`-or-`failed` on the record rather than silence.
- `--confirm-destructive` is required when the proposal carries no rollback. Use it only when the operator
  asked for that exact change.
- After a successful apply the instance is probed again and the receipt (algorithm, rows affected,
  duration) is journaled.

## 5. Report

Quote: what changed, the tier it was proven at, the rows affected, the duration, and the reference. Then:

```bash
noxis show-proposal <pr> --json    # status applied, with the receipt
noxis events <pr> --json           # the whole story, in order
```

## Failure

| Answer | Meaning | Do |
| --- | --- | --- |
| `conflict` | the proposal is not `approved`, or already applied | re-read it; do not force a second apply |
| `forbidden` | the token lacks `approve` or `apply` | stop: ask the operator who should decide |
| apply `failed` | the engine refused at execution time | the record says so; read `events`, then propose a fix |
| probe failed after apply | the statement landed; the connection did not answer | not a failed apply — report both facts separately |

## Never

- Never approve your own proposal, and never apply one whose rehearsal is `failed`.
- Never apply a proposal whose `schema_diff` you cannot explain line by line.
- Never `--confirm-destructive` without the operator's words in the conversation.
- Never delete or edit anything to "clean up" — proposals and the journal are permanent on purpose.
