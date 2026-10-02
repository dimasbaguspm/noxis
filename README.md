# noxis

Personal multi-machine infra control plane. One Go binary: a **master** exposes a
REST API and publishes desired-state events to **slave** workers, which converge
their node's slice by managing Docker containers (postgres, mysql, mariadb,
mongo, redis, elasticsearch). Slave workers receive events over one internal
bidirectional gRPC stream per node — no brokers, no consensus.

## Quickstart

```bash
# master (REST API + internal transport)
export NOXIS_LISTEN=:9000 NOXIS_GRPC=:9100 NOXIS_DB=./noxis.db
./noxis master

# slave worker
export NOXIS_GRPC=master-host:9100 NOXIS_NODE=zeus
./noxis slave

# CLI hits the REST API
export NOXIS_MASTER=http://master-host:9000
./noxis assign zeus redis@7:6380 mongo@8      # custom host port, multi-engine
./noxis assign zeus redis@8                   # version bump -> worker recreates
./noxis status
./noxis query zeus postgres "SELECT 1"        # SQL read, routed over the stream
./noxis sync zeus                             # manual reconcile (fallback)
```

## REST API (frontend-facing)

| Method | Path | Body | Result |
|---|---|---|---|
| GET | `/v1/status` | — | `{nodes:[{node,state,detail,ts}]}` |
| GET | `/v1/nodes/{node}` | — | `{node,engines:[...]}` |
| PUT | `/v1/nodes/{node}` | `{engines:[{name,type,version,ports,env}]}` | 204, publishes to the node's stream |
| POST | `/v1/nodes/{node}/query` | `{engine,sql}` | `{output}` |

```bash
curl -X PUT localhost:9000/v1/nodes/zeus \
  -d '{"engines":[{"name":"postgres","type":"postgres","version":"16-alpine"}]}'
curl -X POST localhost:9000/v1/nodes/zeus/query \
  -d '{"engine":"postgres","sql":"SELECT 1"}'
```

CORS is open for dev frontends.

## Configuration (env only, no defaults)

| Variable | Needed by | Meaning |
|---|---|---|
| `NOXIS_MASTER` | CLI/frontend | REST base URL of the master |
| `NOXIS_GRPC` | slave, master | internal master↔slave transport address |
| `NOXIS_NODE` | slave | this node's id |
| `NOXIS_LISTEN` | master | REST listen address |
| `NOXIS_DB` | master | sqlite path |
| `NOXIS_OTEL_COLLECTOR` | all (optional) | OTLP endpoint, enables trace+metric export |
| `NOXIS_OTEL_NAME` | all (optional) | service name reported to OTEL |

Missing variables fail loudly per command.

## Layout

```
cmd/noxis/               bootstrap (main.go) + commands/ one file per verb
internal/entity/         domain models, Config (env struct), App
internal/interfaces/     seams (SpecRepository, StatusRepository)
internal/repository/     sqlite impls + grpc/ (slave transport) + rest/ (public API)
internal/features/doorman/   reconcile.go (decisions) + runtimes/ (docker + engines)
internal/common/logging  slog JSON + OTEL; logs carry trace_id/span_id from ctx
```

## Dev

```bash
make proto   # regenerate gRPC code after control.proto changes
make build
make test
```