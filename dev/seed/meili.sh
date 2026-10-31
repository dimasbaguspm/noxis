#!/bin/sh
# The demo index: a few documents the console can search, and one a reviewed delete can target.
set -e
HOST="${1:-meilisearch}"
PORT="${2:-7700}"
KEY="${3:-noxis}"
api() { curl -sS -H "Authorization: Bearer $KEY" -H 'Content-Type: application/json' "$@"; }
# A filter needs a filterable attribute, and the server applies settings as a task, so both calls are fire-and-forget
# here: the console's first query waits for the index to settle anyway.
api -X PATCH "http://$HOST:$PORT/indexes/demo/settings" -d '{"filterableAttributes":["user","role","team"]}' >/dev/null
api -X PUT "http://$HOST:$PORT/indexes/demo/documents" -d '[
  {"id":"ada","user":"ada","role":"admin","team":"platform"},
  {"id":"grace","user":"grace","role":"viewer","team":"analytics"},
  {"id":"alan","user":"alan","role":"analyst","team":"analytics"}
]' >/dev/null
echo "seeded ${HOST}:${PORT}"
