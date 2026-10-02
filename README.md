# noxis

Personal multi-machine infra control plane. One Go binary: a **master** exposes a
REST API and publishes desired-state events to **slave** workers over Kafka
(apache/kafka + franz-go). Consumers use durable groups, so a restarted
worker resumes from its last offset — no reconnect/replay loops. The engine
manager (doorman) is paused for the single-machine homelab trial: slaves
currently register, consume their topics, ack specs and heartbeat status —
no Docker dependency. The CLI is one-shot only: register once, run daemons.

## Quickstart

```bash
# master daemon
export NOXIS_LISTEN=:9000 NOXIS_BROKER=localhost:9092 NOXIS_DB=./noxis.db
noxis run master

# on each machine: register the identity once (writes ~/.config/noxis/identity.json)
export NOXIS_MASTER=http://master-host:9000 NOXIS_BROKER=localhost:9092
noxis register slave zeus

# run the daemon — reads the identity, needs NO env
# (run under systemd/pm2; it blocks in the foreground)
noxis run slave zeus

# dev stack with docker compose (master + one slave worker)
make dev
```

## REST API (frontend-facing)

Routes are per-feature modules under `/v1/nodes/{nodeId}/{feature}/*` plus health:

| Method | Path | Result |
|---|---|---|
| GET | `/healthz` | liveness |
| GET | `/v1/status` | `{nodes:[{node,state,detail,ts}]}` (reported + registered) |
| GET | `/v1/nodes/{node}/spec` | `{node,engines:[...]}` |
| PUT | `/v1/nodes/{node}/spec` | 204, publishes to the node's stream |
| POST | `/v1/nodes/{node}/doorman/query` | `{output}` |
| POST | `/v1/nodes/{node}/register` | 204, registers a slave node |

```bash
curl -X PUT localhost:9000/v1/nodes/zeus/spec \
  -d '{"engines":[{"name":"postgres","type":"postgres","version":"16-alpine"}]}'
curl -X POST localhost:9000/v1/nodes/zeus/query -d '{"engine":"postgres","sql":"SELECT 1"}'
```

CORS is open for dev frontends.

## Configuration (env only, no defaults)

| Variable | Needed by | Meaning |
|---|---|---|
| `NOXIS_MASTER` | register | REST base URL of the master |
| `NOXIS_BROKER` | master, register | Kafka broker address (kept in identity) |
| `NOXIS_TOPIC_PREFIX` | master, register | Kafka topic prefix, e.g. `noxis.` (kept in identity) |
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
  bootstrap.go       env -> config -> logging -> root CLI (run/register)
  master/            server.go (daemon), routes.go, middleware.go
    modules/         one file per feature: health, status, spec, query, nodes
  slave/             daemon.go (worker loop), register.go, identity.go
internal/services/       connections: sqlite open+ping, grpc dial, grpc/ transport
internal/entity/         domain models, Config (env struct)
internal/interfaces/     seams (Spec/Status/Node repositories)
internal/repository/     sqlite data layer only (command/query)
internal/features/doorman/   reconcile.go (decisions) + runtimes/ (docker + engines)
internal/common/logging     slog JSON + OTEL; ctx-aware trace_id/span_id
docker/               docker-compose.yml (dev stack) + db.sqlite (dev store)
```

## Dev

```bash
make proto    # regenerate gRPC code after control.proto changes
make swagger  # regenerate doc/swagger.yaml|json from handler annotations
make test
make build
make dev      # docker compose: master + slave worker, stores sqlite under docker/db.sqlite
make down
```

The API spec is annotated on the handlers (`cmd/noxis/master/controller/*`);
`make swagger` regenerates `doc/swagger.yaml|json` as a Postman/import
artifact — the master does not serve it.