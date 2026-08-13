# Development

## Local checks

```sh
gofmt -w internal cmd
go test ./...
go vet ./...
docker build --tag incident-command-lab:dev .
docker compose up --build
```

The default Go server uses deterministic in-memory adapters when no connection
URLs are set. This is the fastest path for unit and HTTP contract tests.
Compose sets `DATABASE_URL` and `NATS_URL`, so the running gateway uses the
PostgreSQL transaction and JetStream publisher/worker. The schema in
`deploy/postgres` is applied at database startup and is the durable outbox
contract.

## Fault drill

Enable one fault at a time with `POST /ops/faults`. Create a reservation with
an `Idempotency-Key`, then inspect `/ops/state`. Create an incident and fetch
its evidence. Repeat a request with the same key to verify exactly one domain
reservation. `backlog` remains retryable until the third processing attempt;
`dependency` compensates stock; `database` fails closed; `bad-release` is
recorded in incident evidence.

## Cloud boundary

`terraform/azure` requires an explicitly selected subscription and protected
GitHub OIDC environment. Workflows only run on `workflow_dispatch`, use a
concurrency lock, and require the `cloud-acceptance` environment for apply,
smoke and destroy. They must not be run from pull requests. Capture only
sanitized outputs and destroy the resource group after smoke tests.
