# [chore]: boot the acceptance engines with testcontainers

Plan issue #764, branch `chore/764-testcontainers`.

## What problem does this solve?

`@slow` booted its engines from a hand-written `docker compose` file with fixed host ports. That was a second thing to keep correct, it collided with anything else on 33306/33307, and two suites could not run at once. It also could not prove what the scenarios actually need: an account that can authenticate.

## What is the user facing?

`UAT_SLOW=1 make uat-slow` starts MySQL 8.4 and MariaDB 11.4 through testcontainers, on ports the runtime picks, seeds them, and removes them afterwards. `make uat` and `make test` are unchanged and still need no docker.

## How will I solve it?

`test/uat/engines.go` builds both containers from `testcontainers-go` (generic container, one dependency), waits for the port, then — because a listening port is not a server that answers — retries as **root** to create the table the scenarios propose against, grant the rehearsal privilege, and finally retries as **the account the steps use**. Addresses travel to the steps as `UAT_<ENGINE>_HOST` / `UAT_<ENGINE>_PORT`, so a scenario registers a target wherever the runtime put it. `compose.yml` and `seed.sql` are deleted.

## Done when

- [x] `UAT_SLOW=1 make uat-slow` is green with no compose file in the tree
- [x] Two slow runs do not collide: the ports are per-run
- [x] The suite waits for the account to authenticate, not just for a port
- [x] `make test` and `make uat` still need no docker

## Risks & notes

- One more dependency in `go.mod`. The `@fast` path never imports it: the container code lives in `test/uat`, which the fast run still builds but never starts.
- The first slow run of a session pulls two images; after that it is ~16s.
