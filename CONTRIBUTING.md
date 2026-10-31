# Contributing

Thanks for looking. noXis is one Go binary with a React console embedded in it, and one SQLite file for its own state — small enough to hold in your head, strict enough to stay that way.

New here? Read [the README](README.md) first for what the thing does, then [AGENTS.md](AGENTS.md) for the repo map.

## Get it running

```bash
make dev postgres     # noxis + postgres, seeded, console on http://localhost:9000
make web              # build the console into the binary's embed dir
make test             # go test ./...            (~30s)
make lint             # golangci-lint v2         (~60s)
make uat              # Gherkin over the real stack (needs docker)
```

The dev stack signs in as `root@noxis.local` / `noxis-admin` and registers the instance it seeded, so you can click through without setting anything up. `make dev mongodb`, `make dev redis` and friends start one target; `make dev all` starts every one; bare `make dev` prints the list.

The console has its own checks, run from `apps/web`:

```bash
pnpm typecheck && pnpm lint && pnpm format:check && pnpm build
```

→ Every target and switch: [docs/agent/infrastructure.md](docs/agent/infrastructure.md) · the seeded stack: [docker-compose.yml](docker-compose.yml), [dev/seed](dev/seed)

## The loop

Every change lands through the tracker, never as an uncommitted blob on `master`.

1. **Epic** — one issue describing the outcome: what problem it solves, what the user sees, how it will be solved, when it is done, and the risks. The template is [`.github/ISSUE_TEMPLATE/ticket.md`](.github/ISSUE_TEMPLATE/ticket.md).
2. **Tickets** — split the epic into vertical slices: each cuts every layer, is demoable alone, and finishes in a session. Show the list before creating it.
3. **Branch and pull request per ticket** — one branch, one PR, `Closes #<n>` on the last line. Title: `[ai:<y|n>][#<ticket>] <type>: <subject>`. The body follows [`.github/PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md).
4. **Review before merge** — the reviewer checks the ticket's *Done when* list against the diff, with `make lint`, `make test`, `make uat` and the web checks green.
5. **Record** — a shipped plan gets a snapshot under [`docs/plan/`](docs/plan), added to its index.

Stacked work is fine: branch each ticket off its parent's branch, and open the PR against that parent. Merge bottom-up.

→ Commit style, PR body and review checklist: [docs/agent/review.md](docs/agent/review.md)

## Conventions that bite

- **One use case = one file = one `Serve`.** A use case never calls another; shared steps are package helpers.
- **Console state lives in a hook.** A component with three or more `useState` calls moves that state into `apps/web/src/hooks/` and keeps only JSX and event wiring. The vendored primitives are exempt.
- **The URL is the truth** in the console: driver in the path, instance in the query string. A page that keeps selection in React state is a bug waiting to be shared as a link.
- **No env reads outside `apps/api/internal/common/config`**, and no hardcoded drivers, capabilities or statuses — they live in `apps/api/internal/constants`.
- **Test doubles are generated**: change an interface in `internal/interfaces` and run `make mocks`, then commit the result.
- **Regenerate, don't hand-edit**: `make swagger` after route or annotation changes (`openapi.yaml|json`), and the web client reads those files.
- **The registry schema is reset-only.** `schemaTemplate` is canonical and versioned; no runtime migrations. An old store fails with an instruction.
- **Changes are artifacts.** Nothing deletes a recorded change; a failed submission is returned, not stored.
- **No new dependency for a few lines.** Prefer the standard library, the platform, then what is already installed.

→ Why each of these exists: [docs/agent/architecture.md](docs/agent/architecture.md) · [docs/agent/database.md](docs/agent/database.md)

## Where things live

| Path | What it owns |
| --- | --- |
| `apps/api/cmd/noxis/` | process entry, routes, middlewares |
| `apps/api/cmd/noxis/server/` | the composition root: wiring, the route table, middlewares |
| `apps/api/internal/usecases/` | one package per domain, one file per use case |
| `apps/api/internal/modules/` | one package per engine driver, plus the registry |
| `apps/api/internal/repositories/` | persistence — SQLite is the only state |
| `apps/api/internal/interfaces/` | the seams: repositories, modules, capability interfaces |
| `apps/api/test/uat/` | the Gherkin acceptance suite |
| `apps/web/src/` | the console: pages, components, hooks, `constants/routes.ts` |
| `dev/seed/` | what the dev stack seeds per target |

→ The full map, with the dependency direction: [AGENTS.md](AGENTS.md) · [docs/agent/architecture.md](docs/agent/architecture.md#c4-ish-layering)

## Getting unstuck

- `make dev <target>` refuses to start? Check `docker compose version` — the stack uses profiles and `required: false`.
- A web change does not show up in the binary? The console is embedded: run `make web`.
- `make uat` needs the whole stack and about eight minutes; run it before opening a PR that touches the API. It reuses the previous image unless you set `UAT_BUILD=1`, so after a branch switch a failure may be a stale binary rather than your code.
- Anything the docs disagree with the code about is a documentation bug — please open the issue.
