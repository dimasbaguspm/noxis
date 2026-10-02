# noxis — plan v1

Personal multi-machine infra control plane. Go, single binary, role from CLI flag.

## Locked decisions (don't revisit without a reason)

1. **Transport:** master exposes a REST API (frontend/CLI); internal master↔slave events flow over one bidirectional gRPC `Watch` stream per node — pushes on change, full replay on (re)connect, query round-trips ride the same stream. No Kafka, no Redis, no broker.
2. **Topology:** 1 master (durable desired-state + CLI) → N slaves (local reconcilers over Docker).
3. **Consensus:** none. Manual `--promote` failover. Raft only if auto-failover becomes real need.
4. **Storage on master:** SQLite. No database to manage your database manager.
5. **Engines:** containers, pinned images, one machine may run many (Docker separation). Engine = one small interface, `sqlEngine` base covers pg/mysql/maria, `mongoEngine` separate.
6. **Backup before any version bump. Manual apply per node. Auto-apply deferred.**
7. Replication payload = desired-state spec, never DB data. Managed DBs stay local to their node.

## Shape

```
cmd/noxis/
  main.go            thin entry
  bootstrap.go       env -> config -> logging -> app -> root CLI (run/register/status)
  master/            server.go + routes.go + controller.go + middleware.go
  slave/             daemon.go (worker loop) + register.go + identity.go
internal/entity/         domain models + Config (env struct, go-validator) + App
internal/interfaces/     seams (Spec/Status/Node repositories)
internal/repository/     sqlite impls + grpc/ (internal slave transport)
internal/features/doorman/   reconcile.go (decisions) + runtimes/ (docker + engines)
internal/features/ucm/      versioned config store on the changelog (next)
```

CLI is one-shot: `run master`, `run slave <id>` (reads identity, zero env),
`register slave <name>` (identity + master registry), `status`. The master's
REST API is the frontend surface; master registration/offline nodes merge
into `/v1/status`.

Engines nest under doorman, not beside it: elasticsearch is not a feature,
it's a Doorman-managed engine. Every function takes ctx first; logs
re-emit trace/span ids from the context so log-trace-metric share identity.

Storage decision: sqlite holds only the assignment record (what should run),
the status echo (what slaves report running), and the changelog (audit/oplog).
Actual runtime lives on the machine (docker ps), read fresh on every
reconcile — no duplicated state, no double-apply by design.

## Phases

- [x] 0 Scaffold — `noxis master` / `slave` / `assign` / `status` / `sync` CLI
- [x] 1 Core transport — Watch stream push + replay on reconnect, GetDesiredState fallback, ReportStatus. Verified: master death → slaves survive, engines keep running → restart → reconnected + converged
- [x] 2 Doorman engines — postgres/mysql/mariadb/mongo/redis/elastic descriptors + docker driver (CLI shell-out, node-labeled isolation)
- [x] 3 Desired-state converge — assign → push → slave start/upgrade/remove; node-scoped listing stops cross-node theft on shared docker hosts; port conflicts surface as clear per-node errors; custom host ports `TYPE@VER:HOSTPORT`; config drift (version/port/env) recreates via the stamped `noxis.desired` label
- [x] 3.5 Read queries — `query NODE ENGINE "SQL"` routes over the bidirectional Watch stream (no slave listeners, no addr registration); slave executes via the engine's own SQL client with its creation credentials; SQL engines only
- [ ] 4 UCM — versioned config entities, `get/set/history` verbs, oplog replication to slaves
- [ ] 5 Ops — backup-before-upgrade, `apply --node X` manual gate, `--promote`, systemd units for agents
- [ ] 6 Polish (only if needed) | mTLS, web UI

Known ceilings (ponytail): proto supports `ports`/`env` overrides but CLI `assign` doesn't set them yet; containers recreate on image change only — env/port edits need manual `docker rm -f` + `sync`.

## Non-goals

- Multi-DC, multi-master, auto-failover, audit webhooks, Kubernetes. Never / when a real trigger appears.

## Risks

- **Data loss on upgrade** — mitigated by phase 5 backup gate, never skipped.
- **Scope creep** — this stays a handful of packages; each new service = thin layer on core, not a new framework.
- **Two-node raft temptation** — reject; 2-node consensus is ceremony, manual promote wins.