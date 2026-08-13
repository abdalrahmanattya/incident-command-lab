#!/usr/bin/env sh
set -eu
base=${BASE_URL:-http://localhost:8080}
curl --fail --silent "$base/v1/catalogue" >/dev/null
curl --fail --silent -X POST "$base/ops/faults" -H 'content-type: application/json' -d '{"fault":"dependency","enabled":true}' >/dev/null
curl --fail --silent -X POST "$base/ops/incidents" -H 'content-type: application/json' -d '{"title":"Dependency drill","severity":"SEV2"}' >/dev/null
curl --fail --silent -X POST "$base/ops/faults" -H 'content-type: application/json' -d '{"fault":"dependency","enabled":false}' >/dev/null
echo 'fault drill passed'
