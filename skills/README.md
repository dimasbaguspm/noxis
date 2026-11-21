# Skills

Ready-to-copy skills for agents that operate noXis. They are versioned with the code, so a skill and the
CLI it describes never drift apart.

Each directory holds one `SKILL.md` in the [Agent Skills](https://code.claude.com/docs/en/skills) format: YAML
frontmatter (`name`, `description`) and a markdown body. That format is read by pi, Claude Code, Cursor and
anything else that loads a skills directory — copy the folder, not the file, and the skill travels.

| Skill | For |
| --- | --- |
| [`noxis-setup`](noxis-setup/SKILL.md) | first install: ask the operator which capabilities you may hold, then register a target |
| [`noxis-read`](noxis-read/SKILL.md) | explore instances, schemas and tables; run `SELECT`/`EXPLAIN`/`SHOW`; look up changes |
| [`noxis-propose`](noxis-propose/SKILL.md) | write a DDL or DML change **and its rollback**, and read the rehearsal that judges it |
| [`noxis-review`](noxis-review/SKILL.md) | decide on a proposal, apply it, and report what the evidence said |

## Copy them

```bash
# pi
cp -r skills/noxis-* ~/.pi/agent/skills/

# Claude Code
cp -r skills/noxis-* ~/.claude/skills/

# one project only (shared with the team, like the repo's own .cursor/rules)
mkdir -p .claude/skills && cp -r skills/noxis-* .claude/skills/
```

An agent that only reads and proposes needs `noxis-read` and `noxis-propose`. `noxis-review` is for a
human's token — an agent holding `approve`/`apply` is an agent reviewing itself, which is the thing noXis
exists to prevent.

## Prerequisites, once

```bash
# signed in, token proved against the server before it is stored (0600)
noxis auth signin <secret> --api-url https://api.example.com

# or CI: no stored profile
export API_URL=https://api.example.com
export API_TOKEN=<secret>
```

Every skill assumes `noxis` is on `PATH`. `noxis describe` is always the first call in a new environment: it
joins the running server's OpenAPI with your scopes and prints what you may actually do.

## Conventions every skill follows

- `--json` on every command when a machine reads the answer.
- A failure is `{"error":{"code","message","status"}}` on **stderr** with a non-zero exit. `code: "client"`
  means the request never left the machine — that is the caller's bug, not the server's.
- Never retry a `propose` without the same `--idempotency-key`; with it, a retry answers the same proposal.
- Nothing in these skills runs DDL directly against a database. `propose` → `approve` → `apply` is the only
  path, and the rehearsal is the evidence.
