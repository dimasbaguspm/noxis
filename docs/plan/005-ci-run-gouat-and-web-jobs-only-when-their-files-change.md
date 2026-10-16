# ci: run Go/UAT and web jobs only when their files change

- **Plan id:** issue 5
- **Area:** CI
- **Status:** completed 2026-10-07
- **Issue:** [#5](https://github.com/dimasbaguspm/noxis/issues/5)
- **Pull request:** [#6](https://github.com/dimasbaguspm/noxis/pull/6)
- **Commit:** [5f291f3](https://github.com/dimasbaguspm/noxis/commit/5f291f34f06990fb24f0705c2a068ab1ea8a8d8c) — `[ai:y][#5] ci: split go and web jobs behind a path filter (#6)`

## Plan

## Context

`ci.yml` currently runs **one sequential pipeline** for every PR and every push to `master`:

```
lint → docker build → go unit → web lint+build → uat
```

Every stage runs regardless of what actually changed.

## Problem

- A web-only change (`web/**`) still runs golangci-lint, the Go unit suite, the Docker build and the full UAT stack (kafka + master + slave).
- A Go-only change still runs the pnpm lint + build.
- This wastes runner minutes and makes feedback slow for the change that was actually made.

## Proposal

Split the single `pipeline` job into **two independent jobs** gated by a changed-path filter:

| Job | Runs when | Steps |
|-----|-----------|-------|
| `go` | Go / test / docker / make / CI files changed | lint → docker build → go unit → uat |
| `web` | `web/**` changed | pnpm install → pnpm lint → pnpm build |

Use [`dorny/paths-filter`](https://github.com/dorny/paths-filter) to compute the filters on both `pull_request` and `push` (a native workflow-level `paths:` cannot gate individual jobs, and would also block PRs that touch only one side).

### Filter definition

```yaml
go:   & go
  - 'cmd/**'
  - 'internal/**'
  - 'test/**'
  - 'go.mod'
  - 'go.sum'
  - '.golangci.yml'
  - 'Makefile'
  - 'Dockerfile'
  - 'docker/**'
  - '.github/workflows/**'
web:
  - 'web/**'
```

### Proposed `ci.yml`

```yaml
name: ci

on:
  push:
    branches: [master]
  pull_request:

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true

jobs:
  changes:
    runs-on: ubuntu-latest
    outputs:
      go:  ${{ steps.filter.outputs.go }}
      web: ${{ steps.filter.outputs.web }}
    steps:
      - uses: actions/checkout@v4
      - uses: dorny/paths-filter@v3
        id: filter
        with:
          filters: |
            go:
              - 'cmd/**'
              - 'internal/**'
              - 'test/**'
              - 'go.mod'
              - 'go.sum'
              - '.golangci.yml'
              - 'Makefile'
              - 'Dockerfile'
              - 'docker/**'
              - '.github/workflows/**'
            web:
              - 'web/**'

  go:
    needs: changes
    if: needs.changes.outputs.go == 'true'
    runs-on: ubuntu-latest
    timeout-minutes: 30
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true
      - name: lint
        run: go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...
      - name: build (Dockerfile)
        run: docker build -t noxis:ci .
      - name: unit tests
        run: go test -count=1 $(go list ./... | grep -v /test/uat)
      - name: uat (real stack via REST)
        run: make uat

  web:
    needs: changes
    if: needs.changes.outputs.web == 'true'
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v4
        with:
          version: 10
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: pnpm
          cache-dependency-path: web/pnpm-lock.yaml
      - run: pnpm install --frozen-lockfile
        working-directory: web
      - run: pnpm lint
        working-directory: web
      - run: pnpm build
        working-directory: web
```

## Notes / trade-offs

- **Docker build** stays in the `go` job. Its first stage builds `web/`, but a web-only change is already covered by the `web` job, so we do not double-run it. If we later want the container image validated on web changes too, gate the docker step on `go || web`.
- **`.github/workflows/**`** is in the `go` filter so a CI edit re-runs the Go side; consider adding it to `web` too if the web job is the one being edited.
- **Required status checks**: if branch protection requires the old `pipeline` check, update it after merge, otherwise PRs will hang waiting on a check that no longer exists.
- If *nothing* matches (docs-only), no job runs — intended.

## Acceptance criteria

- [ ] Web-only PR runs only the `web` job (no Go/UAT/Docker).
- [ ] Go-only PR runs only the `go` job (no pnpm).
- [ ] PR touching both runs both jobs.
- [ ] Push to `master` applies the same path filtering.
- [ ] Branch protection required checks updated to the new job names.

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
