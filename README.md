# noxis

Personal multi-machine infra control plane. One Go binary: a **master** exposes a
REST API and publishes desired-state events to **slave** workers, which converge
their node's slice by managing Docker containers (postgres, mysql, mariadb,
mongo, redis, elasticsearch). Workers receive events over one internal
bidirectional gRPC stream per node — no brokers, no consensus. The CLI is
one-shot only: register once, run daemons, check status.

## Quickstart

```bash
# master daemon
export NOXIS_LISTEN=:9000 NOXIS_GRPC=:9100 NOXIS_DB=./noxis.db
noxis run master

# on each machine: register the identity once (writes ~/.config/noxis/identity.json)
export NOXIS_MASTER=http://master-host:9000 NOXIS_GRPC=master-host:9100
noxis register slave zeus

# run the daemon — reads the identity, needs NO env
# (run under systemd/pm2; it blocks in the foreground)
noxis run slave zeus

# status from anywhere
export NOXIS_MASTER=http://master-host:9000
noxis status
```

## REST API (frontend-facing)

| Method | Path | Body | Result |
|---|---|---|---|
| GET | `/v1/status` | — | `{nodes:[{node,state,detail,ts}]}` (reported + registered) |
| GET | `/v1/nodes/{node}` | — | `{node,engines:[...]}` |
| PUT | `/v1/nodes/{node}` | `{engines:[{name,type,version,ports,env}]}` | 204, publishes to the node's stream |
| POST | `/v1/nodes/{node}/query` | `{engine,sql}` | `{output}` |
| POST | `/v1/nodes/{node}/register` | — | 204, registers a slave node |

```bash
curl -X PUT localhost:9000/v1/nodes/zeus \
  -d '{"engines":[{"name":"postgres","type":"postgres","version":"16-alpine"}]}'
curl -X POST localhost:9000/v1/nodes/zeus/query -d '{"engine":"postgres","sql":"SELECT 1"}'
```

CORS is open for dev frontends.

## Configuration (env only, no defaults)

| Variable | Needed by | Meaning |
|---|---|---|
| `NOXIS_MASTER` | status, register | REST base URL of the master |
| `NOXIS_GRPC` | register, master | internal master↔slave transport address |
| `NOXIS_LISTEN` | master | REST listen address |
| `NOXIS_DB` | master | sqlite path |
| `NOXIS_OTEL_COLLECTOR` | all (optional) | OTLP endpoint, enables trace+metric export |
| `NOXIS_OTEL_NAME` | all (optional) | service name reported to OTEL |

`noxis run slave` reads the identity file (`~/.config/noxis/identity.json`) and
needs no environment. Missing variables fail loudly per command.

## Layout

```
cmd/noxis/
  main.go            thin entry
  bootstrap.go       env -> config -> logging -> app -> root CLI (run/register/status)
  http.go            REST client for the CLI
  master/            serve.go (daemon), routes.go, controller.go, middleware.go (CORS/log)
  slave/             daemon.go (worker loop), register.go, identity.go
internal/entity/         domain models, Config (env struct), App
internal/interfaces/     seams (Spec/Status/Node repositories)
internal/repository/     sqlite impls + grpc/ (internal slave transport)
internal/features/doorman/   reconcile.go (decisions) + runtimes/ (docker + engines)
internal/common/logging     slog JSON + OTEL; ctx-aware trace_id/span_id
```

## Dev

```bash
make proto   # regenerate gRPC code after control.proto changes
make build
make test
```