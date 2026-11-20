# [plan]: one binary, a contract, and a console that ships with it

Plan issue #783, closed 2026-11-21. Seven tickets, each its own branch and pull request.

## What problem does this solve?

v2 shipped two binaries (`noxisd`, `noxis`) and a hand-rolled HTTP stack nobody outside the repo could read. A deployment needed two images or a shell script; an integrator wrote their own client because the API had no machine-readable contract; a new user saw a list of commands with no welcome; the environment carried knobs (`LOG_LEVEL`, `LOG_FORMAT`, `SECRET_KEY_PREVIOUS`) that existed only to serve a compatibility never shipped; observability stopped at the route, so a slow rehearsal was one opaque span; and the console in `web/` had no way to reach someone who only had the binary.

## What shipped

| Ticket | What landed | PR |
| --- | --- | --- |
| #784 | One binary, three faces: `noxis http serve`, `noxis web serve`, and the CLI. The transport moved to `internal/httpapi`; `cmd/noxis` holds the only `main` and the composition root. Distroless image, rootless, read-only, `/data` owned by the runtime user, `HEALTHCHECK` answered by the binary itself. 23.5 MB. | #791 |
| #788 | `LOG_LEVEL`, `LOG_FORMAT` and `SECRET_KEY_PREVIOUS` are gone: JSON at Info, one sealing key, and a row sealed with another one refuses with the instruction to re-enter the password. | #792 |
| #787 | A bare `noxis` greets: three faces, the resolved session, the next four commands. `noxis help` and `noxis ?` print one grouped map. | #793 |
| #785 | `.golangci.yml`: bodyclose, errorlint, gocritic, gosec, misspell, noctx, prealloc, revive, rowserrcheck, sqlclosecheck, unconvert, wastedassign, whitespace, plus gofumpt and goimports. 37 findings, all real: three `%w` that were `%s`, a table snapshot that leaked rows and never checked `Err()`. | #794 |
| #786 | `/openapi.json` and `/openapi.yaml`: OpenAPI 3.1, every operation, `x-scopes` per route, JSON derived from the YAML once. A test walks the route table and fails on drift. | #795 |
| #789 | `noxis web serve --port 3000 --api …` serves `internal/web/dist` from the binary, with a committed placeholder that names the API and says what to do; SPA fallback, 405 on writes, no traversal. | #796 |
| #790 | A span per layer: 19 handlers, 17 use cases, 15 SQLite stores, the client, and the four engine methods from #767. Nested, and a recorder test proves it. | #797 |

## Where this plan deviated

- **The console is embedded, and `AGENTS.md` no longer says it never is.** The v1 rule ("if you find `go:embed` of a built console, that is v1 residue") is now the opposite decision: `internal/web/dist` ships inside the binary, a real build deletes `NOT_BUILT`, and the placeholder is what an unbuilt console serves. The record is in `AGENTS.md` itself.
- **`internal/domain` has no span.** The epic asked for "every layer"; the architecture forbids the one layer that would make that literal. The span set stops at the layers that may import OTEL, and `test/arch` keeps it that way.
- **The console is a placeholder, not a product.** #789 was about the shipping mechanism — embed, serve, port, fallback, honesty about being unbuilt — and the React app itself remains unbuilt. Saying so is the point.
- **`cmd/noxis` coverage dropped from 66% to 60%** when the server loop moved into it, and the floor was lowered with it. That is the one place this plan let a number fall, and it is recorded here rather than hidden.

## Risks that turned out to matter

- Moving the transport out of `package main` touched every path in `AGENTS.md`, the README, `docs/agent/*` and the UAT harness. The docs were part of the ticket, not an afterthought.
- A strict linter on a small tree still found defects a review had passed three times: the `%s` wraps, and rows closed only on the happy path.
- `GOTOOLCHAIN` bit the tree twice: `go test -cover` needs the toolchain the `go.mod` names, and CI's `setup-go` version has to agree with it.
- Removing the previous key makes rotation a password re-entry. That is the honest cost, and the refusal says it.

## Done when

- [x] `noxis http serve`, `noxis web serve` and the CLI commands are one binary, and `make build` produces it
- [x] The image is rootless, read-only and as small as a Go binary plus a CA bundle can be (23.5 MB)
- [x] `make lint` runs a documented strict policy and the tree passes it
- [x] `/openapi.json` and `/openapi.yaml` describe every route in the table, proven by a test
- [x] A human who types `noxis` knows what to do next
- [x] `LOG_LEVEL`, `LOG_FORMAT` and `SECRET_KEY_PREVIOUS` appear nowhere in the tree
- [x] `noxis web serve --port 3333` answers the console from the binary
- [x] A trace of one apply shows the handler, the use case, the store and the engine
