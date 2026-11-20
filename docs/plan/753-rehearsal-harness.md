# [chore]: wire the @slow acceptance harness

Closes the open `@slow` item on #751 and #752. Part of #749.

## What problem does this solve?

Two tickets shipped with their `Done when` half-met: the engine work was verified by hand against containers, but the suite never booted them, so a live probe and a live read were not regression-guarded. A scenario that only exists in a pull request description is not evidence.

## What is the user facing?

`UAT_SLOW=1 make uat-slow` boots MySQL 8.4 and MariaDB 11.4, runs the `@fast` scenarios plus the `@slow` ones, and removes the containers and their volumes afterwards. `make uat` still runs only `@fast`, so the default loop stays fast and docker-free.

## How will I solve it?

`test/uat/compose.yml` declares the two engines on fixed ports with healthchecks. `TestMain` runs `docker compose up -d --wait` when `UAT_SLOW=1`, and `docker compose down -v` on the way out even after a failure. The runner filters by tag: `~@slow` by default, `@slow` for the slow run, so the two paths never mix. Steps register a container as a target, probe it, read from it, and prove a write is still refused against a *reachable* engine.

## Done when

- [x] `UAT_SLOW=1 make uat-slow` boots both engines, runs the slow scenarios and cleans up
- [x] A slow run proves a probe, a read and a refused write against real containers
- [x] `make uat` is unchanged for everyday work: fast, no docker

## Risks & notes

- CI still runs only `@fast`. Adding a docker-enabled job is a separate decision: it makes every push wait for two containers, and the runner's docker is not the operator's docker.
- The compose file is test-only infrastructure and lives beside the steps that name it, not in `dev/`.
