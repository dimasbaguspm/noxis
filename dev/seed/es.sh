#!/bin/sh
# The demo index: a few documents the console can query, and one a reviewed delete can target.
set -e
HOST="${1:-elasticsearch}"
PORT="${2:-9200}"
api() { curl -sS -X "${2:-GET}" "http://$HOST:$PORT$1" "${@:3}"; }
# One document per request by id, so a repeat run overwrites instead of duplicating.
api /demo/_doc/ada -X PUT -H 'Content-Type: application/json' -d '{"user":"ada","role":"admin","team":"platform"}' >/dev/null
api /demo/_doc/grace -X PUT -H 'Content-Type: application/json' -d '{"user":"grace","role":"viewer","team":"analytics"}' >/dev/null
api /demo/_doc/alan -X PUT -H 'Content-Type: application/json' -d '{"user":"alan","role":"analyst","team":"analytics"}' >/dev/null
api /demo/_refresh -X POST >/dev/null
echo "seeded ${HOST}:${PORT}"
