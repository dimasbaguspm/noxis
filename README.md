# noxis

Personal multi-machine infra control plane. One Go binary; a master keeps
durable desired state, slaves converge their node's slice of it by managing
Docker containers (postgres, mysql, mariadb, mongo, redis, elasticsearch).

Transport is gRPC server-streaming: each slave opens a `Watch` stream to the
master, the master pushes changes and replays the full spec on (re)connect.
No brokers, no Kafka/Redis/RabbitMQ, no consensus — failover is manual.

## Quickstart

```bash
# master machine
export NOXIS_LISTEN=:9000 NOXIS_DB=./noxis.db
./noxis master

# slave machine
export NOXIS_MASTER=master-host:9000 NOXIS_NODE=zeus
./noxis slave

# from anywhere with NOXIS_MASTER set
./noxis assign zeus redis@7 mongo@8        # declarative desired state
./noxis assign zeus redis@7:6380           # custom host port (container side stays default)
./noxis assign zeus redis@7:6380 redis@7:6381   # same type twice, auto-suffixed names
./noxis assign zeus redis@8                # version bump -> slave recreates
./noxis status                             # reported runtime state per node
./noxis sync zeus                          # manual reconcile (fallback)
./noxis query zeus postgres "SELECT 1"     # read query, routed over the watch stream (SQL engines)
```

## Configuration (env only, no defaults)

| Variable | Needed by | Meaning |
|---|---|---|
| `NOXIS_MASTER` | slave, assign, status, sync | master address |
| `NOXIS_NODE` | slave | this node's id |
| `NOXIS_LISTEN` | master | master listen address |
| `NOXIS_DB` | master | sqlite path |
| `NOXIS_OTEL_COLLECTOR` | all (optional) | OTLP endpoint, enables trace+metric export |
| `NOXIS_OTEL_NAME` | all (optional) | service name reported to OTEL |

Missing variables fail loudly per command. No secret defaults.

## Layout

```
cmd/noxis/               bootstrap in main.go, commands/ one file per verb
internal/common/logging  slog JSON + OTEL; logs carry trace_id/span_id from ctx
internal/entity/         domain models + Config (env struct) + App container
internal/interfaces/     seams (SpecRepository, StatusRepository)
internal/repository/     sqlite impls + grpc/ (control.proto, server, client)
internal/features/doorman/reconcile.go    diff/apply decisions
internal/features/doorman/runtimes/       docker driver + per-engine recipes
```

Containers drift-recreate on any desired-state change (version, port, env),
compared via a `noxis.desired` label stamped at creation — no inspect parsing.

## Dev

```bash
make proto   # regenerate gRPC code after control.proto changes
make build
make test
```