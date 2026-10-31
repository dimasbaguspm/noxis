#!/bin/sh
# The demo database: one CouchDB database with a few documents the console can render.
set -e
HOST="${1:-couchdb}"
PORT="${2:-5984}"
USER="${3:-admin}"
PASS="${4:-noxis}"
api() { curl -sS -u "$USER:$PASS" "http://$HOST:$PORT$1" "${@:2}"; }
# A second run finds the database and every document already there, so both are left alone.
api /demo -X PUT >/dev/null
api /demo/ada -X PUT -H 'Content-Type: application/json' -d '{"user":"ada","role":"admin","team":"platform"}' >/dev/null
api /demo/grace -X PUT -H 'Content-Type: application/json' -d '{"user":"grace","role":"viewer","team":"analytics"}' >/dev/null
api /demo/alan -X PUT -H 'Content-Type: application/json' -d '{"user":"alan","role":"analyst","team":"analytics"}' >/dev/null
echo "seeded ${HOST}:${PORT}"
