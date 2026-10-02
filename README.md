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
./noxis assign zeus redis@8                # version bump -> slave recreates
./noxis status                             # reported runtime state per node
./noxis sync zeus                          # manual reconcile (fallback)
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
cmd/noxis/          bootstrap: env -> config struct -> logger -> commands
internal/common/    config (env struct), logging (slog JSON + OTEL)
internal/entity/    domain models
internal/interfaces/   seams (SpecRepository, StatusRepository)
internal/repository/   sqlite implementations
internal/core/      control.proto, gRPC server + converters
internal/features/  doorman: engine registry + docker reconciler
```

## Dev

```bash
make proto   # regenerate gRPC code after control.proto changes
make build
make test
```