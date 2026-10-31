#!/bin/sh
# The demo keyspace: a few shapes the console can render, with and without a TTL.
set -e
HOST="${1:-redis}"
PORT="${2:-6379}"
cli() { redis-cli -h "$HOST" -p "$PORT" "$@"; }
cli SET session:ada '{"user":"ada","role":"admin"}' >/dev/null
cli EXPIRE session:ada 3600 >/dev/null
cli SET session:grace '{"user":"grace","role":"viewer"}' >/dev/null
cli SET cache:homepage '<h1>noXis</h1>' EX 600 >/dev/null
cli RPUSH queue:emails 'ada@example.com' 'grace@example.com' >/dev/null
cli HSET user:alan name 'Alan Turing' role analyst >/dev/null
cli SET counter:visits 42 >/dev/null
echo "seeded ${HOST}:${PORT}"
