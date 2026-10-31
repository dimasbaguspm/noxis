#!/bin/sh
# Registers the seeded targets in noXis, so the console opens with instances instead of an empty list.
set -e
API="http://noxis:9000/external/v1"

TOKEN=$(curl -sf -X POST "$API/auth/login" -H 'content-type: application/json' \
  -d "{\"email\":\"${NOXIS_ADMIN_EMAIL}\",\"password\":\"${NOXIS_ADMIN_PASSWORD}\"}" \
  | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
if [ -z "$TOKEN" ]; then
  echo "seed: could not sign in as ${NOXIS_ADMIN_EMAIL}"
  exit 1
fi

register() {
  code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$API/engine/$1/connections" \
    -H "authorization: Bearer $TOKEN" -H 'content-type: application/json' -d "$2")
  case "$code" in
    2*) echo "seed: registered $1" ;;
    409) echo "seed: $1 already registered" ;;
    *) echo "seed: $1 refused with $code" ;;
  esac
}

for driver in $SEED_DRIVERS; do
  case "$driver" in
    postgres)  register postgres  '{"name":"demo-postgres","driver":"postgres","host":"db","port":5432,"user":"postgres","password":"noxis","database":"demo"}' ;;
    mysql)     register mysql     '{"name":"demo-mysql","driver":"mysql","host":"mysql","port":3306,"user":"root","password":"noxis","database":"shop"}' ;;
    mariadb)   register mariadb   '{"name":"demo-mariadb","driver":"mariadb","host":"mariadb","port":3306,"user":"root","password":"noxis","database":"shop"}' ;;
    cockroach) register cockroach '{"name":"demo-cockroach","driver":"cockroach","host":"cockroach","port":26257,"user":"root","database":"demo"}' ;;
    redis)     register redis     '{"name":"demo-redis","driver":"redis","host":"redis","port":6379,"database":"0"}' ;;
    valkey)    register valkey    '{"name":"demo-valkey","driver":"valkey","host":"valkey","port":6379,"database":"0"}' ;;
    dragonfly) register dragonfly '{"name":"demo-dragonfly","driver":"dragonfly","host":"dragonfly","port":6379,"database":"0"}' ;;
    couchdb)   register couchdb   '{"name":"demo-couchdb","driver":"couchdb","host":"couchdb","port":5984,"user":"admin","password":"noxis","database":"demo"}' ;;
    elasticsearch) register elasticsearch '{"name":"demo-elasticsearch","driver":"elasticsearch","host":"elasticsearch","port":9200,"database":"demo"}' ;;
    meilisearch) register meilisearch '{"name":"demo-meilisearch","driver":"meilisearch","host":"meilisearch","port":7700,"password":"noxis","database":"demo"}' ;;
    mongo)     register mongo     '{"name":"demo-mongo","driver":"mongo","host":"mongo","port":27017,"user":"root","password":"noxis","database":"demo"}' ;;
    *) echo "seed: unknown driver $driver" ;;
  esac
done
