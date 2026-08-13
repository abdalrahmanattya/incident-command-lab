#!/usr/bin/env sh
set -eu
base=${BASE_URL:-http://localhost:8080}
curl_auth() { if [ -n "${SMOKE_BEARER:-}" ]; then curl --fail --silent -H "Authorization: Bearer $SMOKE_BEARER" "$@"; else curl --fail --silent "$@"; fi; }
curl_auth "$base/healthz" >/dev/null
if [ -n "${REQUIRE_SMOKE_BEARER:-}" ] && [ -z "${SMOKE_BEARER:-}" ]; then echo 'SMOKE_BEARER is required for protected cloud smoke' >&2; exit 1; fi
set_fault() { if [ -n "${SMOKE_BEARER:-}" ]; then curl --fail --silent -X POST "$base/ops/faults" -H "Authorization: Bearer $SMOKE_BEARER" -H 'content-type: application/json' -d "$1" >/dev/null; else curl --fail --silent -X POST "$base/ops/faults" -H 'content-type: application/json' -d "$1" >/dev/null; fi; }
set_fault '{"fault":"dependency","enabled":true}'
comp=$(curl_auth -X POST "$base/v1/reservations" -H 'content-type: application/json' -H 'idempotency-key: smoke-compensation' -d '{"customer_id":"smoke","product_id":"concert","quantity":1}')
printf '%s' "$comp" | grep -q 'COMPENSATED'
set_fault '{"fault":"dependency","enabled":false}'
body=$(curl_auth -X POST "$base/v1/reservations" -H 'content-type: application/json' -H 'idempotency-key: smoke-1' -d '{"customer_id":"smoke","product_id":"concert","quantity":1}')
printf '%s' "$body" | grep -q 'CONFIRMED'
repeat=$(curl_auth -X POST "$base/v1/reservations" -H 'content-type: application/json' -H 'idempotency-key: smoke-1' -d '{"customer_id":"smoke","product_id":"concert","quantity":1}')
printf '%s' "$repeat" | grep -q 'CONFIRMED'
incident=$(curl_auth -X POST "$base/ops/incidents" -H 'content-type: application/json' -d '{"title":"Smoke dependency drill","severity":"SEV2"}')
id=$(printf '%s' "$incident" | sed -n 's/.*"ID":"\([^"]*\)".*/\1/p')
[ -n "$id" ]
curl_auth "$base/ops/incidents/$id/evidence" | grep -q "$id"
curl_auth -X POST "$base/ops/incidents/$id/analyze" | grep -q 'advisory'
echo "smoke passed: compensation, idempotency, incident analysis ($id)"
